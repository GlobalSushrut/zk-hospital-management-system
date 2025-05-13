package interop

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"
)

// HL7Message represents an HL7 message
type HL7Message struct {
	Segments []string
}

// HL7Delimiters defines the delimiters used in an HL7 message
type HL7Delimiters struct {
	Field     rune
	Component rune
	Repeat    rune
	Escape    rune
	Subcomp   rune
}

// DefaultDelimiters returns the default HL7 delimiters
func DefaultDelimiters() HL7Delimiters {
	return HL7Delimiters{
		Field:     '|',
		Component: '^',
		Repeat:    '~',
		Escape:    '\\',
		Subcomp:   '&',
	}
}

// HL7Client represents a client for sending and receiving HL7 messages
type HL7Client struct {
	host       string
	port       int
	timeout    time.Duration
	conn       net.Conn
	mutex      sync.Mutex
	delimiters HL7Delimiters
}

// NewHL7Client creates a new HL7 client
func NewHL7Client(host string, port int) *HL7Client {
	return &HL7Client{
		host:       host,
		port:       port,
		timeout:    30 * time.Second,
		delimiters: DefaultDelimiters(),
	}
}

// Connect connects to the HL7 server
func (c *HL7Client) Connect() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Close existing connection if any
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	// Connect with timeout
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", c.host, c.port), c.timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to HL7 server: %w", err)
	}

	c.conn = conn
	return nil
}

// Close closes the connection
func (c *HL7Client) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

// SendMessage sends an HL7 message and receives the acknowledgment
func (c *HL7Client) SendMessage(msg *HL7Message) (*HL7Message, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.conn == nil {
		return nil, errors.New("not connected to HL7 server")
	}

	// MLLP wrapping
	data := []byte("\x0B") // VT (vertical tab)
	data = append(data, []byte(msg.String())...)
	data = append(data, []byte("\x1C\x0D")...) // FS CR

	// Set deadline for write and read
	deadline := time.Now().Add(c.timeout)
	if err := c.conn.SetDeadline(deadline); err != nil {
		return nil, fmt.Errorf("failed to set deadline: %w", err)
	}

	// Send message
	if _, err := c.conn.Write(data); err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	// Read response
	reader := bufio.NewReader(c.conn)
	var respData bytes.Buffer
	inMessage := false

	for {
		b, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		if b == 0x0B { // VT - Start of message
			inMessage = true
			continue
		}

		if b == 0x1C { // FS - End of message
			inMessage = false
			break
		}

		if inMessage {
			respData.WriteByte(b)
		}
	}

	// Parse response
	if respData.Len() == 0 {
		return nil, errors.New("received empty response")
	}

	return ParseHL7(respData.String())
}

// ParseHL7 parses an HL7 message string into a structured HL7Message
func ParseHL7(data string) (*HL7Message, error) {
	if len(data) == 0 {
		return nil, errors.New("empty message")
	}

	// Split message into segments
	segments := strings.Split(data, "\r")
	
	// Clean up segments
	cleanSegments := make([]string, 0, len(segments))
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment != "" {
			cleanSegments = append(cleanSegments, segment)
		}
	}

	if len(cleanSegments) == 0 {
		return nil, errors.New("message contains no segments")
	}

	// Ensure the first segment is MSH
	if !strings.HasPrefix(cleanSegments[0], "MSH") {
		return nil, errors.New("message does not start with MSH segment")
	}

	return &HL7Message{Segments: cleanSegments}, nil
}

// String returns the string representation of an HL7 message
func (m *HL7Message) String() string {
	return strings.Join(m.Segments, "\r") + "\r"
}

// GetSegment gets a segment by its type
func (m *HL7Message) GetSegment(segmentType string) (string, bool) {
	pattern := fmt.Sprintf("^%s[|^]", regexp.QuoteMeta(segmentType))
	regex := regexp.MustCompile(pattern)

	for _, segment := range m.Segments {
		if regex.MatchString(segment) {
			return segment, true
		}
	}

	return "", false
}

// GetAllSegments gets all segments of a specific type
func (m *HL7Message) GetAllSegments(segmentType string) []string {
	pattern := fmt.Sprintf("^%s[|^]", regexp.QuoteMeta(segmentType))
	regex := regexp.MustCompile(pattern)
	
	results := make([]string, 0)
	for _, segment := range m.Segments {
		if regex.MatchString(segment) {
			results = append(results, segment)
		}
	}

	return results
}

// GetValue gets a value from a segment by field positions
func (m *HL7Message) GetValue(segmentType string, field int, component int, subcomponent int) (string, error) {
	segment, found := m.GetSegment(segmentType)
	if !found {
		return "", fmt.Errorf("segment %s not found", segmentType)
	}

	// Parse delimiters from MSH segment if needed
	var delimiters HL7Delimiters
	if segmentType == "MSH" && field > 1 {
		// In MSH, field separator is the 4th character
		if len(segment) < 4 {
			return "", errors.New("invalid MSH segment")
		}
		fieldSep := rune(segment[3])
		
		// Component separator is the 5th character if available
		var compSep, repeatSep, escapeSep, subcompSep rune
		if len(segment) > 4 {
			compSep = rune(segment[4])
		} else {
			compSep = '^'
		}
		
		// Other separators follow if available
		if len(segment) > 5 {
			repeatSep = rune(segment[5])
		} else {
			repeatSep = '~'
		}
		
		if len(segment) > 6 {
			escapeSep = rune(segment[6])
		} else {
			escapeSep = '\\'
		}
		
		if len(segment) > 7 {
			subcompSep = rune(segment[7])
		} else {
			subcompSep = '&'
		}
		
		delimiters = HL7Delimiters{
			Field:     fieldSep,
			Component: compSep,
			Repeat:    repeatSep,
			Escape:    escapeSep,
			Subcomp:   subcompSep,
		}
	} else {
		// Use default delimiters for non-MSH segments
		delimiters = DefaultDelimiters()
	}

	// For MSH segment, field 1 is the field separator itself
	// and field 2 is the encoding characters
	fieldOffset := 0
	if segmentType == "MSH" {
		fieldOffset = -1
	}

	// Split into fields
	fields := strings.Split(segment, string(delimiters.Field))
	adjustedField := field + fieldOffset
	
	if adjustedField < 0 || adjustedField >= len(fields) {
		return "", fmt.Errorf("field %d not found in segment %s", field, segmentType)
	}

	fieldValue := fields[adjustedField]

	// If only field is requested, return the whole field
	if component <= 0 {
		return fieldValue, nil
	}

	// Split into components
	components := strings.Split(fieldValue, string(delimiters.Component))
	if component > len(components) {
		return "", fmt.Errorf("component %d not found in field %d of segment %s", 
			component, field, segmentType)
	}

	componentValue := components[component-1]

	// If only component is requested, return the whole component
	if subcomponent <= 0 {
		return componentValue, nil
	}

	// Split into subcomponents
	subcomponents := strings.Split(componentValue, string(delimiters.Subcomp))
	if subcomponent > len(subcomponents) {
		return "", fmt.Errorf("subcomponent %d not found in component %d of field %d of segment %s", 
			subcomponent, component, field, segmentType)
	}

	return subcomponents[subcomponent-1], nil
}

// AddSegment adds a segment to the message
func (m *HL7Message) AddSegment(segment string) {
	m.Segments = append(m.Segments, segment)
}

// CreateACK creates an acknowledgment message
func (m *HL7Message) CreateACK(ackCode string) (*HL7Message, error) {
	// Get MSH segment
	msh, found := m.GetSegment("MSH")
	if !found {
		return nil, errors.New("MSH segment not found")
	}

	// Parse MSH values
	msgControlID, err := m.GetValue("MSH", 10, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("error getting message control ID: %w", err)
	}

	sendingApp, err := m.GetValue("MSH", 3, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("error getting sending application: %w", err)
	}

	sendingFacility, err := m.GetValue("MSH", 4, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("error getting sending facility: %w", err)
	}

	receivingApp, err := m.GetValue("MSH", 5, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("error getting receiving application: %w", err)
	}

	receivingFacility, err := m.GetValue("MSH", 6, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("error getting receiving facility: %w", err)
	}

	// Create MSH segment for ACK
	// Note: In MSH segment, the first field is actually the field separator itself
	mshParts := strings.SplitN(msh, "|", 3)
	if len(mshParts) < 3 {
		return nil, errors.New("invalid MSH segment format")
	}

	// Use the same field separator and encoding characters
	fieldSep := string(msh[3])
	encodingChars := mshParts[1]

	timestamp := time.Now().Format("20060102150405")

	// Create ACK message
	ackMSH := fmt.Sprintf("MSH%s%s%s%s%s%s%s%s%s%s%s%s%s%s%s%s%s",
		fieldSep, encodingChars,
		fieldSep, receivingApp,
		fieldSep, receivingFacility,
		fieldSep, sendingApp,
		fieldSep, sendingFacility,
		fieldSep, timestamp,
		fieldSep, "ACK",
		fieldSep, msgControlID,
		fieldSep, "P")

	ackMSA := fmt.Sprintf("MSA%s%s%s%s",
		fieldSep, ackCode,
		fieldSep, msgControlID)

	ack := &HL7Message{
		Segments: []string{ackMSH, ackMSA},
	}

	return ack, nil
}

// HL7Server represents an HL7 MLLP server
type HL7Server struct {
	port       int
	listener   net.Listener
	handler    func(*HL7Message) (*HL7Message, error)
	shutdown   chan struct{}
	wg         sync.WaitGroup
	timeout    time.Duration
}

// NewHL7Server creates a new HL7 server
func NewHL7Server(port int, handler func(*HL7Message) (*HL7Message, error)) *HL7Server {
	return &HL7Server{
		port:     port,
		handler:  handler,
		shutdown: make(chan struct{}),
		timeout:  30 * time.Second,
	}
}

// Start starts the HL7 server
func (s *HL7Server) Start() error {
	var err error
	s.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to start HL7 server: %w", err)
	}

	// Handle connections in a goroutine
	s.wg.Add(1)
	go s.acceptConnections()

	return nil
}

// Stop stops the HL7 server
func (s *HL7Server) Stop() error {
	close(s.shutdown)
	
	if s.listener != nil {
		err := s.listener.Close()
		s.wg.Wait()
		return err
	}
	
	return nil
}

// acceptConnections accepts incoming connections
func (s *HL7Server) acceptConnections() {
	defer s.wg.Done()

	for {
		select {
		case <-s.shutdown:
			return
		default:
			// Accept with timeout
			s.listener.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Second))
			conn, err := s.listener.Accept()
			
			if err != nil {
				if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
					// Timeout, check for shutdown and continue
					continue
				}
				// Other error, log and continue
				fmt.Printf("Error accepting connection: %v\n", err)
				continue
			}

			// Handle the connection in a new goroutine
			s.wg.Add(1)
			go s.handleConnection(conn)
		}
	}
}

// handleConnection processes an incoming connection
func (s *HL7Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	// Set connection deadline
	conn.SetDeadline(time.Now().Add(s.timeout))

	// Read message with MLLP framing
	reader := bufio.NewReader(conn)
	var msgData bytes.Buffer
	inMessage := false

	for {
		b, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Printf("Error reading from connection: %v\n", err)
			return
		}

		if b == 0x0B { // VT - Start of message
			inMessage = true
			continue
		}

		if b == 0x1C { // FS - End of message
			inMessage = false
			break
		}

		if inMessage {
			msgData.WriteByte(b)
		}
	}

	// Parse the received message
	msg, err := ParseHL7(msgData.String())
	if err != nil {
		fmt.Printf("Error parsing HL7 message: %v\n", err)
		return
	}

	// Process the message with the handler
	var response *HL7Message
	if s.handler != nil {
		response, err = s.handler(msg)
		if err != nil {
			fmt.Printf("Error handling HL7 message: %v\n", err)
			
			// Create error ACK
			response, err = msg.CreateACK("AE") // Application Error
			if err != nil {
				fmt.Printf("Error creating error ACK: %v\n", err)
				return
			}
		}
	} else {
		// Default behavior: send ACK
		response, err = msg.CreateACK("AA") // Application Accept
		if err != nil {
			fmt.Printf("Error creating ACK: %v\n", err)
			return
		}
	}

	// Send response with MLLP framing
	respStr := response.String()
	respData := []byte("\x0B") // VT
	respData = append(respData, []byte(respStr)...)
	respData = append(respData, []byte("\x1C\x0D")...) // FS CR

	if _, err := conn.Write(respData); err != nil {
		fmt.Printf("Error writing response: %v\n", err)
	}
}

// HL7MessageBuilder helps build HL7 messages
type HL7MessageBuilder struct {
	segments []string
	delimiters HL7Delimiters
}

// NewHL7MessageBuilder creates a new HL7 message builder
func NewHL7MessageBuilder() *HL7MessageBuilder {
	return &HL7MessageBuilder{
		segments: make([]string, 0),
		delimiters: DefaultDelimiters(),
	}
}

// WithMSH adds an MSH segment
func (b *HL7MessageBuilder) WithMSH(sendingApp, sendingFacility, receivingApp, receivingFacility, 
	messageType, messageControlID string) *HL7MessageBuilder {
	
	timestamp := time.Now().Format("20060102150405")
	
	// Create encoding characters string
	encodingChars := string([]rune{
		b.delimiters.Component,
		b.delimiters.Repeat,
		b.delimiters.Escape,
		b.delimiters.Subcomp,
	})
	
	msh := fmt.Sprintf("MSH%c%s%c%s%c%s%c%s%c%s%c%s%c%s%c%s%c%s",
		b.delimiters.Field, encodingChars,
		b.delimiters.Field, sendingApp,
		b.delimiters.Field, sendingFacility,
		b.delimiters.Field, receivingApp,
		b.delimiters.Field, receivingFacility,
		b.delimiters.Field, timestamp,
		b.delimiters.Field, messageType,
		b.delimiters.Field, messageControlID,
		b.delimiters.Field, "P")
	
	b.segments = append(b.segments, msh)
	return b
}

// AddSegment adds a raw segment
func (b *HL7MessageBuilder) AddSegment(segment string) *HL7MessageBuilder {
	b.segments = append(b.segments, segment)
	return b
}

// AddPID adds a PID (Patient Identification) segment
func (b *HL7MessageBuilder) AddPID(patientID, patientName, dob, gender string) *HL7MessageBuilder {
	// Split the patient name into parts
	nameParts := strings.Split(patientName, " ")
	lastName := ""
	firstName := ""
	
	if len(nameParts) > 0 {
		lastName = nameParts[len(nameParts)-1]
	}
	
	if len(nameParts) > 1 {
		firstName = strings.Join(nameParts[:len(nameParts)-1], " ")
	}
	
	pid := fmt.Sprintf("PID%c1%c%s%c%c%s%c%s%c%c%s%c%c%s",
		b.delimiters.Field,
		b.delimiters.Field, patientID,
		b.delimiters.Field,
		b.delimiters.Field, fmt.Sprintf("%s%c%s", lastName, b.delimiters.Component, firstName),
		b.delimiters.Field,
		b.delimiters.Field,
		b.delimiters.Field,
		b.delimiters.Field, dob,
		b.delimiters.Field,
		b.delimiters.Field, gender)
	
	b.segments = append(b.segments, pid)
	return b
}

// AddPV1 adds a PV1 (Patient Visit) segment
func (b *HL7MessageBuilder) AddPV1(patientClass, assignedLocation, attendingDoctor string) *HL7MessageBuilder {
	pv1 := fmt.Sprintf("PV1%c1%c%s%c%s%c%c%c%c%c%s",
		b.delimiters.Field,
		b.delimiters.Field, patientClass,
		b.delimiters.Field, assignedLocation,
		b.delimiters.Field,
		b.delimiters.Field,
		b.delimiters.Field,
		b.delimiters.Field,
		b.delimiters.Field, attendingDoctor)
	
	b.segments = append(b.segments, pv1)
	return b
}

// AddOBR adds an OBR (Observation Request) segment
func (b *HL7MessageBuilder) AddOBR(setID, placerOrderNumber, fillerOrderNumber, universalServiceID, 
	observationDateTime string) *HL7MessageBuilder {
	
	obr := fmt.Sprintf("OBR%c%s%c%s%c%s%c%s%c%c%c%c%s",
		b.delimiters.Field, setID,
		b.delimiters.Field, placerOrderNumber,
		b.delimiters.Field, fillerOrderNumber,
		b.delimiters.Field, universalServiceID,
		b.delimiters.Field,
		b.delimiters.Field,
		b.delimiters.Field,
		b.delimiters.Field, observationDateTime)
	
	b.segments = append(b.segments, obr)
	return b
}

// AddOBX adds an OBX (Observation Result) segment
func (b *HL7MessageBuilder) AddOBX(setID, valueType, observationID, observationValue, units, 
	referenceRange, abnormalFlags string) *HL7MessageBuilder {
	
	obx := fmt.Sprintf("OBX%c%s%c%s%c%s%c%c%s%c%s%c%s%c%s",
		b.delimiters.Field, setID,
		b.delimiters.Field, valueType,
		b.delimiters.Field, observationID,
		b.delimiters.Field,
		b.delimiters.Field, observationValue,
		b.delimiters.Field, units,
		b.delimiters.Field, referenceRange,
		b.delimiters.Field, abnormalFlags)
	
	b.segments = append(b.segments, obx)
	return b
}

// Build returns the constructed HL7 message
func (b *HL7MessageBuilder) Build() *HL7Message {
	return &HL7Message{
		Segments: b.segments,
	}
}
