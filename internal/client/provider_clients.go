package client

// ProviderClients bundles IAM, Console, and S3 clients for resource access.
type ProviderClients struct {
	IAM     *IAMClient
	Console *ConsoleClient
	S3      *S3Client
}
