// Package thirdpartyaws provides helper functions for uploading files to
// Amazon S3. It supports uploading base64-encoded data (commonly received
// from frontend clients) and automatically detects the file type from the
// base64 data URI prefix, sets the appropriate content type, and generates
// a unique file name using UUID.
package thirdpartyaws

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
)

// FileExtension represents a file extension string (e.g., "jpeg", "png", "pdf").
// It is used internally to determine the correct file name suffix and to map
// to the corresponding MIME content type for S3 uploads.
type FileExtension string

const (
	JPEG FileExtension = "jpeg"
	PNG  FileExtension = "png"
	GIF  FileExtension = "gif"
	BMP  FileExtension = "bmp"
	WEBP FileExtension = "webp"
	TIFF FileExtension = "tiff"
	PDF  FileExtension = "pdf"
	TXT  FileExtension = "txt"
	ZIP  FileExtension = "zip"
	MP4  FileExtension = "mp4"
)

// ContentType represents an HTTP MIME content type string (e.g., "image/jpeg").
// It is used to set the Content-Type header when uploading objects to S3,
// ensuring that browsers and other clients handle the file correctly.
type ContentType string

const (
	JPEGContentType ContentType = "image/jpeg"
	PNGContentType  ContentType = "image/png"
	GIFContentType  ContentType = "image/gif"
	BMPContentType  ContentType = "image/bmp"
	WEBPContentType ContentType = "image/webp"
	TIFFContentType ContentType = "image/tiff"
	PDFContentType  ContentType = "application/pdf"
	TXTContentType  ContentType = "text/plain"
	ZIPContentType  ContentType = "application/zip"
	MP4ContentType  ContentType = "video/mp4"
)

// AwsS3StorageBuilder is an internal builder that holds the metadata required
// to upload a file to S3. It accumulates the file extension, generated file
// name, and the corresponding content type before the actual upload is performed.
type AwsS3StorageBuilder struct {
	fileExtension FileExtension
	fileName      string
	contentType   ContentType
}

// getExtension extracts the file extension from a base64 data URI string.
// It parses the MIME type portion of the data URI (the part before the
// semicolon, e.g., "data:image/jpeg") and maps it to the corresponding
// FileExtension constant.
//
// The input is expected to be a data URI like "data:image/jpeg;base64,/9j/4..."
// or at minimum contain the MIME type prefix "image/jpeg;base64,...".
//
// Returns an error if the MIME type format is invalid or if the MIME subtype
// does not match any supported file extension.
//
// Supported MIME types: jpeg, jpg, png, gif, bmp, webp, tiff, pdf, plain (txt),
// zip, mp4.
func getExtension(base64Data string) (FileExtension, error) {
	mime := strings.Split(base64Data, ";")[0] // Extract MIME type
	mimeParts := strings.Split(mime, "/")
	if len(mimeParts) != 2 {
		return "", fmt.Errorf("Invalid base64 string")
	}

	switch mimeParts[1] {
	case "jpeg", "jpg":
		return JPEG, nil
	case "png":
		return PNG, nil
	case "gif":
		return GIF, nil
	case "bmp":
		return BMP, nil
	case "webp":
		return WEBP, nil
	case "tiff":
		return TIFF, nil
	case "pdf":
		return PDF, nil
	case "plain":
		return TXT, nil
	case "zip":
		return ZIP, nil
	case "mp4":
		return MP4, nil
	default:
		return "", fmt.Errorf("no matching extension found for MIME type %s", mime)
	}
}

// setDefaultFileName generates a unique file name for the upload using UUID v4
// and sets it on the builder. The file name format is "{uuid}.{extension}"
// (e.g., "550e8400-e29b-41d4-a716-446655440000.jpeg"). This ensures every
// uploaded file has a globally unique name, preventing collisions in S3.
func setDefaultFileName(builder *AwsS3StorageBuilder) {
	builder.fileName = fmt.Sprintf("%s.%s", uuid.New().String(), builder.fileExtension)
}

// setDefaultContentType sets the HTTP Content-Type on the builder based on
// its file extension. This mapping ensures that when the file is uploaded to
// S3, it is served with the correct MIME type so browsers and other HTTP
// clients handle the file correctly (e.g., displaying images inline rather
// than downloading them).
//
// Panics if the file extension does not match any known content type. This
// should never happen in practice because getExtension only returns known
// FileExtension constants.
func setDefaultContentType(builder *AwsS3StorageBuilder) {
	switch builder.fileExtension {
	case JPEG:
		builder.contentType = JPEGContentType
	case PNG:
		builder.contentType = PNGContentType
	case GIF:
		builder.contentType = GIFContentType
	case BMP:
		builder.contentType = BMPContentType
	case WEBP:
		builder.contentType = WEBPContentType
	case TIFF:
		builder.contentType = TIFFContentType
	case PDF:
		builder.contentType = PDFContentType
	case TXT:
		builder.contentType = TXTContentType
	case ZIP:
		builder.contentType = ZIPContentType
	case MP4:
		builder.contentType = MP4ContentType
	default:
		panic(fmt.Errorf("no matching content type found for file extension %s", builder.fileExtension))
	}
}

// Upload uploads a base64-encoded file to an AWS S3 bucket and returns the
// public URL of the uploaded object (without query string parameters).
//
// The function performs the following steps:
//  1. Extracts the MIME type from the base64 data URI to determine the file
//     extension (e.g., "image/jpeg" -> "jpeg").
//  2. Generates a unique file name using UUID (e.g., "550e8400-...-.jpeg").
//  3. Sets the appropriate Content-Type header for the S3 object.
//  4. Decodes the base64 payload (everything after the "," in the data URI).
//  5. Creates an AWS session with the provided credentials for the
//     "ap-southeast-1" region.
//  6. Uploads the decoded bytes to S3 at the path "{awsBucketPath}/{fileName}".
//  7. Generates a presigned URL and strips the query parameters to return
//     just the base object URL.
//
// Parameters:
//   - awsAccessKey:       The AWS IAM access key ID for authentication.
//   - awsSecretAccessKey: The AWS IAM secret access key for authentication.
//   - awsBucketName:      The name of the target S3 bucket.
//   - awsBucketPath:      The folder/prefix path within the bucket (e.g., "uploads/images").
//   - src:                The full base64 data URI string, e.g.,
//     "data:image/jpeg;base64,/9j/4AAQ...".
//
// Returns the public URL of the uploaded file, or an error if any step fails.
func Upload(awsAccessKey string, awsSecretAccessKey string, awsBucketName string, awsBucketPath string, src string) (string, error) {
	token := ""

	builder := &AwsS3StorageBuilder{}
	fileExt, err := getExtension(src)
	if err != nil {
		return "", fmt.Errorf("error getting file extension: %s", err)
	}
	builder.fileExtension = fileExt
	setDefaultFileName(builder)
	setDefaultContentType(builder)

	dataIndex := strings.Index(src, ",") + 1
	imageData, err := base64.StdEncoding.DecodeString(src[dataIndex:])
	if err != nil {
		return "", fmt.Errorf("error decoding base64 data: %s", err)
	}

	credential := credentials.NewStaticCredentials(awsAccessKey, awsSecretAccessKey, token)
	_, err = credential.Get()
	if err != nil {
		return "", err
	}

	cfg := aws.NewConfig().WithRegion("ap-southeast-1").WithCredentials(credential)
	svc := s3.New(session.New(), cfg)
	fileBytes := bytes.NewReader(imageData)
	objectKey := fmt.Sprintf("%s/%s", awsBucketPath, builder.fileName)

	params := &s3.PutObjectInput{
		Bucket:        aws.String(awsBucketName),
		Key:           aws.String(objectKey),
		Body:          fileBytes,
		ContentLength: aws.Int64(int64(len(imageData))),
		ContentType:   aws.String(string(builder.contentType)),
	}
	_, err = svc.PutObject(params)
	if err != nil {
		return "", err
	}

	req, _ := svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(awsBucketName),
		Key:    aws.String(objectKey),
	})
	url, err := req.Presign(time.Hour)
	if err != nil {
		return "", fmt.Errorf("error getting URL: %s", err)
	}

	return strings.Split(url, "?")[0], nil
}
