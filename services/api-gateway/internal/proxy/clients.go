package proxy

import "google.golang.org/grpc"

// Clients holds all downstream service client wrappers.
type Clients struct {
	Auth      *AuthClient
	Workspace *WorkspaceClient
	Label     *LabelClient
	Agent     *AgentClient
	Print     *PrintClient
	Product   *ProductClient
	Billing   *BillingClient
}

// NewClients wires all gRPC connections and HTTP clients into typed wrappers.
func NewClients(
	authConn, workspaceConn, labelConn, agentConn *grpc.ClientConn,
	printBaseURL, agentAdminURL, productBaseURL, billingBaseURL string,
) *Clients {
	return &Clients{
		Auth:      NewAuthClient(authConn),
		Workspace: NewWorkspaceClient(workspaceConn),
		Label:     NewLabelClient(labelConn),
		Agent:     NewAgentClientWithAdmin(agentConn, agentAdminURL),
		Print:     NewPrintClient(printBaseURL),
		Product:   NewProductClient(productBaseURL),
		Billing:   NewBillingClient(billingBaseURL),
	}
}
