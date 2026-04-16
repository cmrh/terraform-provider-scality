package client

// ProviderClients bundles both IAM and Console clients for resource access.
type ProviderClients struct {
	IAM     *IAMClient
	Console *ConsoleClient
}
