package mocks

// This file includes go:generate statements for regenerating mocks.

//go:generate mockgen -destination s3client_generated.go -package mocks .. S3Client
