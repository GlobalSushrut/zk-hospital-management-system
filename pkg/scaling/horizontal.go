package scaling

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// LoadBalancer represents a distributed load balancing system
type LoadBalancer struct {
	nodes            map[string]*Node
	nodesMutex       sync.RWMutex
	algorithm        BalancingAlgorithm
	healthCheck      *HealthChecker
	stats            *LoadStats
	serviceDiscovery ServiceDiscoveryProvider
}

// Node represents a server node in the cluster
type Node struct {
	ID            string
	Address       string
	Port          int
	Weight        int       // For weighted algorithms
	IsActive      bool
	LastSeen      time.Time
	Capabilities  []string  // What services this node can handle
	CurrentLoad   int32     // Current request count
	ResponseTimes []time.Duration
	Tags          map[string]string
}

// BalancingAlgorithm defines the algorithm to distribute load
type BalancingAlgorithm string

const (
	// RoundRobin distributes load sequentially across nodes
	RoundRobin BalancingAlgorithm = "round-robin"
	
	// LeastConnections sends requests to the node with the least active connections
	LeastConnections BalancingAlgorithm = "least-connections"
	
	// WeightedRoundRobin distributes load based on node capacity weights
	WeightedRoundRobin BalancingAlgorithm = "weighted-round-robin"
	
	// IPHash maps client IPs to specific nodes for session consistency
	IPHash BalancingAlgorithm = "ip-hash"
)

// LoadStats tracks statistics for the load balancer
type LoadStats struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	AvgResponseTime    int64 // in microseconds
	RequestsPerSecond  float64
	lastUpdated        time.Time
	statsLock          sync.RWMutex
}

// ServiceDiscoveryProvider defines the interface for service discovery
type ServiceDiscoveryProvider interface {
	RegisterNode(node *Node) error
	DeregisterNode(nodeID string) error
	DiscoverNodes(ctx context.Context) ([]*Node, error)
	WatchForChanges(ctx context.Context, callback func([]*Node))
}

// NewLoadBalancer creates a new load balancer with the specified algorithm
func NewLoadBalancer(algorithm BalancingAlgorithm, discoveryProvider ServiceDiscoveryProvider) *LoadBalancer {
	lb := &LoadBalancer{
		nodes:            make(map[string]*Node),
		algorithm:        algorithm,
		stats:            &LoadStats{lastUpdated: time.Now()},
		serviceDiscovery: discoveryProvider,
	}
	
	// Initialize health checker
	lb.healthCheck = NewHealthChecker(lb, 30*time.Second)
	
	return lb
}

// AddNode adds a new node to the load balancer
func (lb *LoadBalancer) AddNode(node *Node) {
	lb.nodesMutex.Lock()
	defer lb.nodesMutex.Unlock()
	
	node.IsActive = true
	node.LastSeen = time.Now()
	lb.nodes[node.ID] = node
	
	// Register with service discovery if provided
	if lb.serviceDiscovery != nil {
		if err := lb.serviceDiscovery.RegisterNode(node); err != nil {
			log.Printf("Failed to register node %s with service discovery: %v", node.ID, err)
		}
	}
}

// RemoveNode removes a node from the load balancer
func (lb *LoadBalancer) RemoveNode(nodeID string) {
	lb.nodesMutex.Lock()
	defer lb.nodesMutex.Unlock()
	
	// Deregister from service discovery if provided
	if lb.serviceDiscovery != nil {
		if err := lb.serviceDiscovery.DeregisterNode(nodeID); err != nil {
			log.Printf("Failed to deregister node %s from service discovery: %v", nodeID, err)
		}
	}
	
	delete(lb.nodes, nodeID)
}

// GetNodes returns all nodes in the load balancer
func (lb *LoadBalancer) GetNodes() map[string]*Node {
	lb.nodesMutex.RLock()
	defer lb.nodesMutex.RUnlock()
	
	// Create a copy of the nodes map to avoid concurrent access issues
	nodes := make(map[string]*Node, len(lb.nodes))
	for id, node := range lb.nodes {
		nodes[id] = node
	}
	
	return nodes
}

// GetNextNode gets the next available node according to the balancing algorithm
func (lb *LoadBalancer) GetNextNode(ctx context.Context, clientIP string, requestCapability string) (*Node, error) {
	lb.nodesMutex.RLock()
	defer lb.nodesMutex.RUnlock()
	
	// Filter active nodes that support the requested capability
	activeNodes := make([]*Node, 0)
	for _, node := range lb.nodes {
		if node.IsActive {
			// Skip nodes that don't have the required capability
			if requestCapability != "" {
				hasCapability := false
				for _, capability := range node.Capabilities {
					if capability == requestCapability {
						hasCapability = true
						break
					}
				}
				if !hasCapability {
					continue
				}
			}
			activeNodes = append(activeNodes, node)
		}
	}
	
	if len(activeNodes) == 0 {
		return nil, fmt.Errorf("no active nodes available for capability: %s", requestCapability)
	}
	
	// Select node based on the chosen algorithm
	var selectedNode *Node
	
	switch lb.algorithm {
	case RoundRobin:
		// Simple round robin - atomically increment and mod
		nextIndex := atomic.AddInt64(&lb.stats.TotalRequests, 1) % int64(len(activeNodes))
		selectedNode = activeNodes[nextIndex]
		
	case LeastConnections:
		// Find node with lowest current load
		minLoad := int32(^uint32(0) >> 1) // Max int32
		for _, node := range activeNodes {
			if node.CurrentLoad < minLoad {
				minLoad = node.CurrentLoad
				selectedNode = node
			}
		}
		
	case WeightedRoundRobin:
		// Weighted selection based on node capacity
		totalWeight := 0
		for _, node := range activeNodes {
			totalWeight += node.Weight
		}
		
		// If no weights are set, fall back to round robin
		if totalWeight == 0 {
			nextIndex := atomic.AddInt64(&lb.stats.TotalRequests, 1) % int64(len(activeNodes))
			selectedNode = activeNodes[nextIndex]
		} else {
			// Pick a random point in the weight distribution
			n := atomic.AddInt64(&lb.stats.TotalRequests, 1) % int64(totalWeight)
			currentWeight := 0
			for _, node := range activeNodes {
				currentWeight += node.Weight
				if int64(currentWeight) > n {
					selectedNode = node
					break
				}
			}
		}
		
	case IPHash:
		// Hash the client IP to consistently select the same node
		if clientIP == "" {
			// Fall back to round robin if no client IP
			nextIndex := atomic.AddInt64(&lb.stats.TotalRequests, 1) % int64(len(activeNodes))
			selectedNode = activeNodes[nextIndex]
		} else {
			// Simple hash function for IP
			hash := 0
			for i := 0; i < len(clientIP); i++ {
				hash = 31*hash + int(clientIP[i])
			}
			if hash < 0 {
				hash = -hash
			}
			selectedNode = activeNodes[hash%len(activeNodes)]
		}
		
	default:
		// Default to round robin
		nextIndex := atomic.AddInt64(&lb.stats.TotalRequests, 1) % int64(len(activeNodes))
		selectedNode = activeNodes[nextIndex]
	}
	
	if selectedNode == nil {
		return nil, fmt.Errorf("failed to select a node with algorithm %s", lb.algorithm)
	}
	
	// Update load tracking
	atomic.AddInt32(&selectedNode.CurrentLoad, 1)
	
	return selectedNode, nil
}

// SetHealthCheck updates the health check interval
func (lb *LoadBalancer) SetHealthCheck(interval time.Duration) {
	lb.healthCheck.SetInterval(interval)
}

// StartHealthCheck starts the health check process
func (lb *LoadBalancer) StartHealthCheck(ctx context.Context) {
	lb.healthCheck.Start(ctx)
}

// StopHealthCheck stops the health check process
func (lb *LoadBalancer) StopHealthCheck() {
	lb.healthCheck.Stop()
}

// ReleaseNode decrements the load counter when a request completes
func (lb *LoadBalancer) ReleaseNode(nodeID string, responseTime time.Duration, success bool) {
	lb.nodesMutex.RLock()
	node, exists := lb.nodes[nodeID]
	lb.nodesMutex.RUnlock()
	
	if !exists {
		return
	}
	
	// Update node stats
	atomic.AddInt32(&node.CurrentLoad, -1)
	
	// Track response time (limit to last 100)
	lb.nodesMutex.Lock()
	node.ResponseTimes = append(node.ResponseTimes, responseTime)
	if len(node.ResponseTimes) > 100 {
		node.ResponseTimes = node.ResponseTimes[len(node.ResponseTimes)-100:]
	}
	lb.nodesMutex.Unlock()
	
	// Update global stats
	lb.stats.statsLock.Lock()
	defer lb.stats.statsLock.Unlock()
	
	if success {
		atomic.AddInt64(&lb.stats.SuccessfulRequests, 1)
	} else {
		atomic.AddInt64(&lb.stats.FailedRequests, 1)
	}
	
	// Update average response time
	atomic.StoreInt64(&lb.stats.AvgResponseTime, 
		(atomic.LoadInt64(&lb.stats.AvgResponseTime)*atomic.LoadInt64(&lb.stats.TotalRequests) + 
			responseTime.Microseconds()) / (atomic.LoadInt64(&lb.stats.TotalRequests) + 1))
	
	// Update requests per second
	now := time.Now()
	duration := now.Sub(lb.stats.lastUpdated).Seconds()
	if duration > 5 { // Only update RPS every 5 seconds
		lb.stats.RequestsPerSecond = float64(atomic.LoadInt64(&lb.stats.TotalRequests)) / duration
		lb.stats.lastUpdated = now
	}
}

// HealthChecker is responsible for checking node health
type HealthChecker struct {
	lb       *LoadBalancer
	interval time.Duration
	stopChan chan struct{}
	ticker   *time.Ticker
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(lb *LoadBalancer, interval time.Duration) *HealthChecker {
	return &HealthChecker{
		lb:       lb,
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

// SetInterval updates the health check interval
func (hc *HealthChecker) SetInterval(interval time.Duration) {
	if hc.ticker != nil {
		hc.ticker.Stop()
		hc.ticker = time.NewTicker(interval)
	}
	hc.interval = interval
}

// Start begins the health checking process
func (hc *HealthChecker) Start(ctx context.Context) {
	hc.ticker = time.NewTicker(hc.interval)
	
	go func() {
		// Perform initial health check
		hc.checkHealth(ctx)
		
		for {
			select {
			case <-hc.ticker.C:
				hc.checkHealth(ctx)
			case <-hc.stopChan:
				hc.ticker.Stop()
				return
			case <-ctx.Done():
				hc.ticker.Stop()
				return
			}
		}
	}()
}

// Stop stops the health checking process
func (hc *HealthChecker) Stop() {
	close(hc.stopChan)
}

// checkHealth checks the health of all nodes
func (hc *HealthChecker) checkHealth(ctx context.Context) {
	hc.lb.nodesMutex.RLock()
	nodes := make([]*Node, 0, len(hc.lb.nodes))
	for _, node := range hc.lb.nodes {
		nodes = append(nodes, node)
	}
	hc.lb.nodesMutex.RUnlock()
	
	var wg sync.WaitGroup
	
	for _, node := range nodes {
		wg.Add(1)
		go func(node *Node) {
			defer wg.Done()
			
			// In a real implementation, this would make an HTTP call to the node's health endpoint
			// Here we simply simulate success/failure based on response time
			
			// Simulate health check
			healthy := true
			
			// For simulation, we'll consider a node unhealthy if its current load is too high
			if node.CurrentLoad > 100 { // Arbitrary threshold
				healthy = false
			}
			
			// Update node status
			hc.lb.nodesMutex.Lock()
			// Only update if the node still exists
			if existingNode, exists := hc.lb.nodes[node.ID]; exists {
				existingNode.IsActive = healthy
				existingNode.LastSeen = time.Now()
			}
			hc.lb.nodesMutex.Unlock()
		}(node)
	}
	
	// Wait for all health checks to complete
	wg.Wait()
	
	// If using service discovery, sync nodes
	if hc.lb.serviceDiscovery != nil {
		// Discover available nodes
		discoveredNodes, err := hc.lb.serviceDiscovery.DiscoverNodes(ctx)
		if err != nil {
			log.Printf("Failed to discover nodes: %v", err)
			return
		}
		
		// Update node list
		hc.lb.nodesMutex.Lock()
		
		// Add new nodes
		for _, node := range discoveredNodes {
			if _, exists := hc.lb.nodes[node.ID]; !exists {
				node.IsActive = true
				node.LastSeen = time.Now()
				hc.lb.nodes[node.ID] = node
			}
		}
		
		// Remove nodes that no longer exist in service discovery
		discoveredNodeMap := make(map[string]bool)
		for _, node := range discoveredNodes {
			discoveredNodeMap[node.ID] = true
		}
		
		for id, node := range hc.lb.nodes {
			if !discoveredNodeMap[id] {
				// If the node hasn't been seen for 2 check intervals, remove it
				if time.Since(node.LastSeen) > 2*hc.interval {
					delete(hc.lb.nodes, id)
				}
			}
		}
		
		hc.lb.nodesMutex.Unlock()
	}
}

// ------------------------
// Auto-Scaling Implementation
// ------------------------

// AutoScaler handles automatic scaling of nodes based on load
type AutoScaler struct {
	lb               *LoadBalancer
	minNodes         int
	maxNodes         int
	targetLoad       float64 // Target CPU/memory/request load percentage
	scaleUpThreshold float64 // When to scale up
	scaleDownThreshold float64 // When to scale down
	cooldownPeriod   time.Duration
	lastScaled       time.Time
	nodeTemplate     *Node
	scaleUpFn        func(int) error
	scaleDownFn      func([]string) error
}

// NewAutoScaler creates a new auto scaler
func NewAutoScaler(lb *LoadBalancer, minNodes, maxNodes int, targetLoad float64) *AutoScaler {
	return &AutoScaler{
		lb:                 lb,
		minNodes:           minNodes,
		maxNodes:           maxNodes,
		targetLoad:         targetLoad,
		scaleUpThreshold:   targetLoad * 1.3,  // Scale up at 30% above target
		scaleDownThreshold: targetLoad * 0.5,  // Scale down at 50% below target
		cooldownPeriod:     3 * time.Minute,
		lastScaled:         time.Now().Add(-10 * time.Minute), // Allow immediate scaling on start
	}
}

// SetScaleFunctions sets the functions used to scale up/down
func (as *AutoScaler) SetScaleFunctions(scaleUp func(int) error, scaleDown func([]string) error) {
	as.scaleUpFn = scaleUp
	as.scaleDownFn = scaleDown
}

// SetNodeTemplate sets the template to use for new nodes
func (as *AutoScaler) SetNodeTemplate(template *Node) {
	as.nodeTemplate = template
}

// Start begins the auto-scaling process
func (as *AutoScaler) Start(ctx context.Context, checkInterval time.Duration) {
	ticker := time.NewTicker(checkInterval)
	
	go func() {
		for {
			select {
			case <-ticker.C:
				as.checkAndScale(ctx)
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

// checkAndScale evaluates current load and scales as needed
func (as *AutoScaler) checkAndScale(ctx context.Context) {
	// Don't scale if in cooldown period
	if time.Since(as.lastScaled) < as.cooldownPeriod {
		return
	}
	
	as.lb.nodesMutex.RLock()
	
	// Count active nodes and calculate average load
	activeNodes := 0
	totalLoad := int32(0)
	nodeIDs := make([]string, 0)
	
	for id, node := range as.lb.nodes {
		if node.IsActive {
			activeNodes++
			totalLoad += node.CurrentLoad
			nodeIDs = append(nodeIDs, id)
		}
	}
	
	as.lb.nodesMutex.RUnlock()
	
	// No active nodes to evaluate
	if activeNodes == 0 {
		// Scale up to minimum if we have scale function
		if as.scaleUpFn != nil && as.minNodes > 0 {
			log.Printf("Auto-scaling: No active nodes, scaling up to minimum %d nodes", as.minNodes)
			as.scaleUpFn(as.minNodes)
			as.lastScaled = time.Now()
		}
		return
	}
	
	// Calculate average load per node
	avgLoad := float64(totalLoad) / float64(activeNodes)
	maxDesiredLoad := float64(100) // Arbitrary max load per node
	currentLoadPercentage := avgLoad / maxDesiredLoad
	
	// Determine if scaling is needed
	if currentLoadPercentage > as.scaleUpThreshold && activeNodes < as.maxNodes {
		// Need to scale up
		// Calculate how many new nodes we need
		targetNodes := int(float64(totalLoad) / (maxDesiredLoad * as.targetLoad))
		if targetNodes <= activeNodes {
			targetNodes = activeNodes + 1 // At least add one node
		}
		if targetNodes > as.maxNodes {
			targetNodes = as.maxNodes
		}
		
		nodesToAdd := targetNodes - activeNodes
		if nodesToAdd > 0 && as.scaleUpFn != nil {
			log.Printf("Auto-scaling: Scaling up from %d to %d nodes (load: %.2f%%)", 
				activeNodes, targetNodes, currentLoadPercentage*100)
			as.scaleUpFn(nodesToAdd)
			as.lastScaled = time.Now()
		}
	} else if currentLoadPercentage < as.scaleDownThreshold && activeNodes > as.minNodes {
		// Need to scale down
		// Calculate target number of nodes
		targetNodes := int(float64(totalLoad) / (maxDesiredLoad * as.targetLoad))
		if targetNodes < as.minNodes {
			targetNodes = as.minNodes
		}
		if targetNodes >= activeNodes {
			return // No need to scale down
		}
		
		// Select nodes to remove
		nodesToRemove := activeNodes - targetNodes
		if nodesToRemove > 0 && as.scaleDownFn != nil {
			// Simple strategy: remove the newest nodes (last in, first out)
			// In a real system, would choose based on resource utilization or other metrics
			nodesToRemoveIDs := nodeIDs[len(nodeIDs)-nodesToRemove:]
			
			log.Printf("Auto-scaling: Scaling down from %d to %d nodes (load: %.2f%%)", 
				activeNodes, targetNodes, currentLoadPercentage*100)
			as.scaleDownFn(nodesToRemoveIDs)
			as.lastScaled = time.Now()
		}
	}
}

// ------------------------
// Cluster API Server Implementation
// ------------------------

// ClusterAPIServer manages the cluster via an HTTP API
type ClusterAPIServer struct {
	lb         *LoadBalancer
	autoScaler *AutoScaler
	port       int
}

// NewClusterAPIServer creates a new cluster API server
func NewClusterAPIServer(lb *LoadBalancer, autoScaler *AutoScaler, port int) *ClusterAPIServer {
	return &ClusterAPIServer{
		lb:         lb,
		autoScaler: autoScaler,
		port:       port,
	}
}

// GetClusterStatus returns the current cluster status
func (cas *ClusterAPIServer) GetClusterStatus() map[string]interface{} {
	cas.lb.nodesMutex.RLock()
	defer cas.lb.nodesMutex.RUnlock()
	
	// Count active nodes
	activeNodes := 0
	for _, node := range cas.lb.nodes {
		if node.IsActive {
			activeNodes++
		}
	}
	
	// Get load stats
	cas.lb.stats.statsLock.RLock()
	statsSnapshot := *cas.lb.stats
	cas.lb.stats.statsLock.RUnlock()
	
	// Build response
	return map[string]interface{}{
		"total_nodes":          len(cas.lb.nodes),
		"active_nodes":         activeNodes,
		"algorithm":            cas.lb.algorithm,
		"total_requests":       atomic.LoadInt64(&statsSnapshot.TotalRequests),
		"successful_requests":  atomic.LoadInt64(&statsSnapshot.SuccessfulRequests),
		"failed_requests":      atomic.LoadInt64(&statsSnapshot.FailedRequests),
		"avg_response_time_us": atomic.LoadInt64(&statsSnapshot.AvgResponseTime),
		"requests_per_second":  statsSnapshot.RequestsPerSecond,
	}
}

// GetNodeList returns the list of all nodes in the cluster
func (cas *ClusterAPIServer) GetNodeList() []*NodeInfo {
	cas.lb.nodesMutex.RLock()
	defer cas.lb.nodesMutex.RUnlock()
	
	nodeInfos := make([]*NodeInfo, 0, len(cas.lb.nodes))
	
	for _, node := range cas.lb.nodes {
		// Calculate average response time
		avgRespTime := int64(0)
		if len(node.ResponseTimes) > 0 {
			total := int64(0)
			for _, rt := range node.ResponseTimes {
				total += rt.Microseconds()
			}
			avgRespTime = total / int64(len(node.ResponseTimes))
		}
		
		nodeInfos = append(nodeInfos, &NodeInfo{
			ID:               node.ID,
			Address:          node.Address,
			Port:             node.Port,
			IsActive:         node.IsActive,
			LastSeen:         node.LastSeen.Format(time.RFC3339),
			CurrentLoad:      atomic.LoadInt32(&node.CurrentLoad),
			AvgResponseTime:  avgRespTime,
			Capabilities:     node.Capabilities,
			Tags:             node.Tags,
		})
	}
	
	return nodeInfos
}

// NodeInfo is a simplified view of a Node for API responses
type NodeInfo struct {
	ID               string            `json:"id"`
	Address          string            `json:"address"`
	Port             int               `json:"port"`
	IsActive         bool              `json:"is_active"`
	LastSeen         string            `json:"last_seen"`
	CurrentLoad      int32             `json:"current_load"`
	AvgResponseTime  int64             `json:"avg_response_time_us"`
	Capabilities     []string          `json:"capabilities,omitempty"`
	Tags             map[string]string `json:"tags,omitempty"`
}

// AddNode adds a new node to the cluster
func (cas *ClusterAPIServer) AddNode(node *Node) error {
	// Validate node data
	if node.ID == "" {
		return fmt.Errorf("node ID cannot be empty")
	}
	if node.Address == "" {
		return fmt.Errorf("node address cannot be empty")
	}
	if node.Port <= 0 {
		return fmt.Errorf("invalid port number")
	}
	
	cas.lb.AddNode(node)
	return nil
}

// RemoveNode removes a node from the cluster
func (cas *ClusterAPIServer) RemoveNode(nodeID string) error {
	cas.lb.nodesMutex.RLock()
	_, exists := cas.lb.nodes[nodeID]
	cas.lb.nodesMutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}
	
	cas.lb.RemoveNode(nodeID)
	return nil
}

// SetNodeActive sets a node's active status
func (cas *ClusterAPIServer) SetNodeActive(nodeID string, active bool) error {
	cas.lb.nodesMutex.Lock()
	defer cas.lb.nodesMutex.Unlock()
	
	node, exists := cas.lb.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}
	
	node.IsActive = active
	return nil
}

// SetNodeTags sets custom tags for a node
func (cas *ClusterAPIServer) SetNodeTags(nodeID string, tags map[string]string) error {
	cas.lb.nodesMutex.Lock()
	defer cas.lb.nodesMutex.Unlock()
	
	node, exists := cas.lb.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}
	
	node.Tags = tags
	return nil
}
