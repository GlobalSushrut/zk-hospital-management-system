package demo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

// Base API URL
const (
	baseURL = "http://localhost:8080"
)

// Patient represents a patient in the healthcare system
type Patient struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Age           int      `json:"age"`
	Gender        string   `json:"gender"`
	Conditions    []string `json:"conditions"`
	Jurisdiction  string   `json:"jurisdiction"`
	ConsentGiven  bool     `json:"consent_given"`
	DataSensitive bool     `json:"data_sensitive"`
}

// Doctor represents a healthcare provider
type Doctor struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Specialty    string   `json:"specialty"`
	Hospital     string   `json:"hospital"`
	Jurisdiction string   `json:"jurisdiction"`
	Roles        []string `json:"roles"`
}

// Document represents a healthcare document
type Document struct {
	ID          string    `json:"id"`
	PatientID   string    `json:"patient_id"`
	DoctorID    string    `json:"doctor_id"`
	Type        string    `json:"type"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
	Sensitivity string    `json:"sensitivity"`
}

// Proof represents a ZK proof
type Proof struct {
	ID           string                 `json:"id"`
	CircuitType  string                 `json:"circuit_type"`
	Proof        string                 `json:"proof"`
	PublicInputs map[string]interface{} `json:"public_inputs"`
	ValidUntil   time.Time              `json:"valid_until"`
}

// AccessRequest represents a healthcare data access request
type AccessRequest struct {
	RequestID     string    `json:"request_id"`
	RequesterID   string    `json:"requester_id"`
	RequesterRole string    `json:"requester_role"`
	PatientID     string    `json:"patient_id"`
	DocumentType  string    `json:"document_type"`
	Purpose       string    `json:"purpose"`
	RequestedAt   time.Time `json:"requested_at"`
	Emergency     bool      `json:"emergency"`
}

// Workflow orchestrates the healthcare data flow simulation
type Workflow struct {
	patients    []Patient
	doctors     []Doctor
	documents   []Document
	proofs      map[string]Proof
	accessLogs  []AccessRequest
	verbose     bool
	successRate int
}

// NewWorkflow creates a new workflow with generated test data
func NewWorkflow(patientCount, doctorCount int, verbose bool) *Workflow {
	rand.Seed(time.Now().UnixNano())
	
	wf := &Workflow{
		patients:    generatePatients(patientCount),
		doctors:     generateDoctors(doctorCount),
		documents:   []Document{},
		proofs:      make(map[string]Proof),
		accessLogs:  []AccessRequest{},
		verbose:     verbose,
		successRate: 0,
	}
	
	return wf
}

// Run executes the healthcare workflow simulation
func (wf *Workflow) Run(ctx context.Context) error {
	log.Println("Starting Healthcare Workflow Simulation")
	
	// Track total operations and successes for calculating success rate
	totalOps := 0
	successOps := 0
	
	// Step 1: Generate medical documents for patients
	log.Println("Step 1: Creating medical documents")
	docs, err := wf.createMedicalDocuments()
	if err != nil {
		return fmt.Errorf("error creating medical documents: %v", err)
	}
	wf.documents = append(wf.documents, docs...)
	totalOps += len(docs)
	successOps += len(docs)
	
	// Step 2: Generate consent proofs for patients
	log.Println("Step 2: Generating patient consent proofs")
	for _, patient := range wf.patients {
		if patient.ConsentGiven {
			success, _ := wf.generateConsentProof(patient)
			totalOps++
			if success {
				successOps++
			}
		}
	}
	
	// Step 3: Simulate cross-jurisdiction access requests
	log.Println("Step 3: Testing cross-jurisdiction access")
	for i := 0; i < 20; i++ {
		// Get random doctor and patient from different jurisdictions
		doctor := wf.getRandomDoctorWithDifferentJurisdiction(wf.patients[rand.Intn(len(wf.patients))].Jurisdiction)
		patient := wf.getRandomPatientFromJurisdiction(doctor.Jurisdiction)
		
		// Generate access request
		success, _ := wf.requestAccess(doctor, patient, "medical_history", "treatment", false)
		totalOps++
		if success {
			successOps++
		}
		
		// Try with emergency flag
		success, _ = wf.requestAccess(doctor, patient, "medical_history", "treatment", true)
		totalOps++
		if success {
			successOps++
		}
	}
	
	// Step 4: Validate various policy scenarios
	log.Println("Step 4: Testing comprehensive policy validations")
	
	// Test standard access scenarios
	log.Println("4.1: Testing standard access scenarios")
	for i := 0; i < 15; i++ {
		doctor := wf.doctors[rand.Intn(len(wf.doctors))]
		patient := wf.patients[rand.Intn(len(wf.patients))]
		
		success, _ := wf.requestAccess(doctor, patient, "medical_history", "treatment", false)
		totalOps++
		if success {
			successOps++
		}
	}
	
	// Test role-based access
	log.Println("4.2: Testing role-based access policies")
	for _, role := range []string{"physician", "nurse", "researcher", "insurance_agent"} {
		doctor := wf.doctors[rand.Intn(len(wf.doctors))]
		doctor.Roles = []string{role}
		patient := wf.patients[rand.Intn(len(wf.patients))]
		
		success, _ := wf.validateRoleBasedAccess(doctor, patient, "medical_history")
		totalOps++
		if success {
			successOps++
		}
	}
	
	// Test different data sensitivity levels
	log.Println("4.3: Testing data sensitivity levels")
	for _, sensitivity := range []string{"low", "medium", "high"} {
		doctor := wf.doctors[rand.Intn(len(wf.doctors))]
		patient := wf.patients[rand.Intn(len(wf.patients))]
		
		success, _ := wf.validateSensitiveDataAccess(doctor, patient, sensitivity)
		totalOps++
		if success {
			successOps++
		}
	}
	
	// Calculate success rate
	wf.successRate = (successOps * 100) / totalOps
	
	// Log final statistics
	log.Printf("Workflow simulation completed:")
	log.Printf("- Total operations: %d", totalOps)
	log.Printf("- Successful operations: %d", successOps)
	log.Printf("- Success rate: %d%%", wf.successRate)
	log.Printf("- Documents generated: %d", len(wf.documents))
	log.Printf("- Proofs generated: %d", len(wf.proofs))
	log.Printf("- Access requests processed: %d", len(wf.accessLogs))
	
	return nil
}

// createMedicalDocuments generates random medical documents for patients
func (wf *Workflow) createMedicalDocuments() ([]Document, error) {
	documents := []Document{}
	
	docTypes := []string{
		"medical_history", "prescription", "lab_result", 
		"imaging_report", "consultation_note", "discharge_summary",
	}
	
	sensitivities := []string{"low", "medium", "high"}
	
	// Create 2-5 documents per patient
	for _, patient := range wf.patients {
		numDocs := rand.Intn(4) + 2
		for i := 0; i < numDocs; i++ {
			doctor := wf.doctors[rand.Intn(len(wf.doctors))]
			docType := docTypes[rand.Intn(len(docTypes))]
			sensitivity := sensitivities[rand.Intn(len(sensitivities))]
			
			// Create document
			doc := Document{
				ID:          uuid.New().String(),
				PatientID:   patient.ID,
				DoctorID:    doctor.ID,
				Type:        docType,
				Content:     fmt.Sprintf("Medical content for patient %s by doctor %s", patient.Name, doctor.Name),
				CreatedAt:   time.Now().Add(-time.Duration(rand.Intn(30)) * 24 * time.Hour),
				Sensitivity: sensitivity,
			}
			
			// Store document in Cassandra
			success := wf.storeDocument(doc)
			if success {
				documents = append(documents, doc)
				if wf.verbose {
					log.Printf("Created document %s of type %s for patient %s", doc.ID, doc.Type, patient.ID)
				}
			}
		}
	}
	
	return documents, nil
}

// storeDocument stores a document in the system
func (wf *Workflow) storeDocument(doc Document) bool {
	endpoint := fmt.Sprintf("%s/document/store", baseURL)
	
	// Create request payload
	payload := map[string]interface{}{
		"doc_type": doc.Type,
		"content":  doc.Content,
		"owner_id": doc.PatientID,
	}
	
	// Send request
	resp, err := sendJSONRequest("POST", endpoint, payload)
	if err != nil {
		if wf.verbose {
			log.Printf("Error storing document: %v", err)
		}
		return false
	}
	
	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated
}

// generateConsentProof creates a ZK proof for patient consent
func (wf *Workflow) generateConsentProof(patient Patient) (bool, Proof) {
	endpoint := fmt.Sprintf("%s/zkcircuit/execute", baseURL)
	
	// Generate unique provider ID 
	providerId := fmt.Sprintf("hospital-%d", rand.Intn(5)+1)
	
	// Create proof input data
	payload := map[string]interface{}{
		"circuit_type": "patient-consent",
		"public_inputs": map[string]interface{}{
			"patient_id":   patient.ID,
			"provider_id":  providerId,
			"data_type":    "medical_records",
		},
		"private_inputs": map[string]interface{}{
			"consent_signature": fmt.Sprintf("sig-%s-%d", patient.ID, time.Now().Unix()),
			"timestamp":         time.Now().Unix(),
			"expiration":        time.Now().Add(30 * 24 * time.Hour).Unix(),
		},
	}
	
	// Send request to generate proof
	resp, err := sendJSONRequest("POST", endpoint, payload)
	if err != nil {
		if wf.verbose {
			log.Printf("Error generating consent proof: %v", err)
		}
		return false, Proof{}
	}
	
	if resp.StatusCode != http.StatusOK {
		if wf.verbose {
			log.Printf("Failed to generate consent proof: status %d", resp.StatusCode)
		}
		return false, Proof{}
	}
	
	// Parse response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		if wf.verbose {
			log.Printf("Error parsing consent proof response: %v", err)
		}
		return false, Proof{}
	}
	
	// Create and store proof
	proof := Proof{
		ID:           uuid.New().String(),
		CircuitType:  "patient-consent",
		Proof:        fmt.Sprintf("%v", result["proof"]),
		PublicInputs: payload["public_inputs"].(map[string]interface{}),
		ValidUntil:   time.Now().Add(30 * 24 * time.Hour),
	}
	
	wf.proofs[patient.ID] = proof
	
	if wf.verbose {
		log.Printf("Generated consent proof for patient %s valid until %s", 
			patient.ID, proof.ValidUntil.Format(time.RFC3339))
	}
	
	return true, proof
}

// requestAccess simulates a healthcare provider requesting access to patient data
func (wf *Workflow) requestAccess(doctor Doctor, patient Patient, recordType, purpose string, emergency bool) (bool, error) {
	endpoint := fmt.Sprintf("%s/policy/validate", baseURL)
	
	// Create policy validation request
	request := map[string]interface{}{
		"requester": map[string]interface{}{
			"id":           doctor.ID,
			"role":         doctor.Roles[0],
			"department":   doctor.Specialty,
			"jurisdiction": doctor.Jurisdiction,
		},
		"subject": map[string]interface{}{
			"id":           patient.ID,
			"record_type":  recordType,
			"sensitivity":  "high",
			"jurisdiction": patient.Jurisdiction,
		},
		"action":      "read",
		"purpose":     purpose,
		"auth_method": "two_factor",
		"emergency":   emergency,
	}
	
	// Log access request
	accessLog := AccessRequest{
		RequestID:     uuid.New().String(),
		RequesterID:   doctor.ID,
		RequesterRole: doctor.Roles[0],
		PatientID:     patient.ID,
		DocumentType:  recordType,
		Purpose:       purpose,
		RequestedAt:   time.Now(),
		Emergency:     emergency,
	}
	wf.accessLogs = append(wf.accessLogs, accessLog)
	
	// Send validation request
	resp, err := sendJSONRequest("POST", endpoint, request)
	if err != nil {
		if wf.verbose {
			log.Printf("Error validating access: %v", err)
		}
		return false, err
	}
	
	// Parse response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		if wf.verbose {
			log.Printf("Error parsing access response: %v", err)
		}
		return false, err
	}
	
	allowed, _ := result["allowed"].(bool)
	reason, _ := result["reason"].(string)
	
	if wf.verbose {
		if allowed {
			log.Printf("Access GRANTED for doctor %s to patient %s records (%s): %s", 
				doctor.ID, patient.ID, recordType, reason)
		} else {
			log.Printf("Access DENIED for doctor %s to patient %s records (%s): %s", 
				doctor.ID, patient.ID, recordType, reason)
		}
	}
	
	return allowed, nil
}

// validateRoleBasedAccess tests role-based access control policies
func (wf *Workflow) validateRoleBasedAccess(doctor Doctor, patient Patient, recordType string) (bool, error) {
	endpoint := fmt.Sprintf("%s/policy/role", baseURL)
	
	// Create role validation request
	request := map[string]interface{}{
		"requester": map[string]interface{}{
			"id":           doctor.ID,
			"role":         doctor.Roles[0],
			"department":   doctor.Specialty,
			"jurisdiction": doctor.Jurisdiction,
		},
		"subject": map[string]interface{}{
			"id":           patient.ID,
			"record_type":  recordType,
			"sensitivity":  "high",
			"jurisdiction": patient.Jurisdiction,
		},
		"action":      "read",
		"purpose":     "treatment",
		"auth_method": "two_factor",
	}
	
	// Send validation request
	resp, err := sendJSONRequest("POST", endpoint, request)
	if err != nil {
		if wf.verbose {
			log.Printf("Error validating role-based access: %v", err)
		}
		return false, err
	}
	
	// Parse response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		if wf.verbose {
			log.Printf("Error parsing role-based access response: %v", err)
		}
		return false, err
	}
	
	allowed, _ := result["allowed"].(bool)
	reason, _ := result["reason"].(string)
	
	if wf.verbose {
		log.Printf("Role-based access for role '%s': %v - %s", doctor.Roles[0], allowed, reason)
	}
	
	return allowed, nil
}

// validateSensitiveDataAccess tests access control for data with different sensitivity levels
func (wf *Workflow) validateSensitiveDataAccess(doctor Doctor, patient Patient, sensitivity string) (bool, error) {
	endpoint := fmt.Sprintf("%s/policy/validate", baseURL)
	
	// Create sensitivity validation request
	request := map[string]interface{}{
		"requester": map[string]interface{}{
			"id":           doctor.ID,
			"role":         "physician",
			"department":   doctor.Specialty,
			"jurisdiction": doctor.Jurisdiction,
		},
		"subject": map[string]interface{}{
			"id":           patient.ID,
			"record_type":  "medical_history",
			"sensitivity":  sensitivity,
			"jurisdiction": patient.Jurisdiction,
		},
		"action":      "read",
		"purpose":     "treatment",
		"auth_method": "two_factor",
	}
	
	// Send validation request
	resp, err := sendJSONRequest("POST", endpoint, request)
	if err != nil {
		if wf.verbose {
			log.Printf("Error validating sensitive data access: %v", err)
		}
		return false, err
	}
	
	// Parse response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		if wf.verbose {
			log.Printf("Error parsing sensitive data access response: %v", err)
		}
		return false, err
	}
	
	allowed, _ := result["allowed"].(bool)
	reason, _ := result["reason"].(string)
	
	if wf.verbose {
		log.Printf("Access for %s sensitivity data: %v - %s", sensitivity, allowed, reason)
	}
	
	return allowed, nil
}

// getRandomDoctorWithDifferentJurisdiction gets a doctor from a different jurisdiction
func (wf *Workflow) getRandomDoctorWithDifferentJurisdiction(patientJurisdiction string) Doctor {
	candidates := []Doctor{}
	for _, doctor := range wf.doctors {
		if doctor.Jurisdiction != patientJurisdiction {
			candidates = append(candidates, doctor)
		}
	}
	
	if len(candidates) == 0 {
		return wf.doctors[rand.Intn(len(wf.doctors))]
	}
	
	return candidates[rand.Intn(len(candidates))]
}

// getRandomPatientFromJurisdiction gets a patient from a specific jurisdiction
func (wf *Workflow) getRandomPatientFromJurisdiction(jurisdiction string) Patient {
	candidates := []Patient{}
	for _, patient := range wf.patients {
		if patient.Jurisdiction == jurisdiction {
			candidates = append(candidates, patient)
		}
	}
	
	if len(candidates) == 0 {
		return wf.patients[rand.Intn(len(wf.patients))]
	}
	
	return candidates[rand.Intn(len(candidates))]
}

// generatePatients creates random patient data
func generatePatients(count int) []Patient {
	patients := make([]Patient, count)
	
	conditions := []string{
		"Hypertension", "Diabetes", "Asthma", "Arthritis", 
		"Depression", "Anxiety", "COPD", "Cancer", "Heart Disease",
	}
	
	jurisdictions := []string{
		"california", "new_york", "texas", "florida", "illinois",
	}
	
	for i := 0; i < count; i++ {
		numConditions := rand.Intn(3)
		patientConditions := []string{}
		for j := 0; j < numConditions; j++ {
			condition := conditions[rand.Intn(len(conditions))]
			patientConditions = append(patientConditions, condition)
		}
		
		patient := Patient{
			ID:            fmt.Sprintf("P%d", 10000+i),
			Name:          fmt.Sprintf("Patient-%d", i),
			Age:           rand.Intn(60) + 18,
			Gender:        []string{"male", "female", "other"}[rand.Intn(3)],
			Conditions:    patientConditions,
			Jurisdiction:  jurisdictions[rand.Intn(len(jurisdictions))],
			ConsentGiven:  rand.Intn(100) < 85, // 85% chance of consent
			DataSensitive: rand.Intn(100) < 30, // 30% chance of sensitive data
		}
		
		patients[i] = patient
	}
	
	return patients
}

// generateDoctors creates random doctor data
func generateDoctors(count int) []Doctor {
	doctors := make([]Doctor, count)
	
	specialties := []string{
		"Cardiology", "Neurology", "Pediatrics", "Oncology", 
		"Radiology", "Internal Medicine", "Surgery", "Psychiatry",
	}
	
	hospitals := []string{
		"General Hospital", "University Medical Center", 
		"Memorial Hospital", "Saint Mary's", "City Medical",
	}
	
	jurisdictions := []string{
		"california", "new_york", "texas", "florida", "illinois",
	}
	
	roles := []string{
		"physician", "specialist", "resident", "attending",
	}
	
	for i := 0; i < count; i++ {
		doctorRoles := []string{roles[rand.Intn(len(roles))]}
		
		doctor := Doctor{
			ID:           fmt.Sprintf("D%d", 10000+i),
			Name:         fmt.Sprintf("Dr. Provider-%d", i),
			Specialty:    specialties[rand.Intn(len(specialties))],
			Hospital:     hospitals[rand.Intn(len(hospitals))],
			Jurisdiction: jurisdictions[rand.Intn(len(jurisdictions))],
			Roles:        doctorRoles,
		}
		
		doctors[i] = doctor
	}
	
	return doctors
}

// Helper function to send JSON requests
func sendJSONRequest(method, url string, payload interface{}) (*http.Response, error) {
	// Marshal payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling JSON: %v", err)
	}
	
	// Create request
	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	
	// Send request
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	return client.Do(req)
}

func main() {
	// Setup logging
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	
	// Parse command line arguments
	verbose := true
	
	// Create and run workflow
	workflow := NewWorkflow(20, 10, verbose)
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	
	// Run the workflow
	if err := workflow.Run(ctx); err != nil {
		log.Fatalf("Error running healthcare workflow: %v", err)
	}
	
	log.Println("Healthcare workflow simulation completed successfully")
}
