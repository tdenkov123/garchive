package validation

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/tdenkov123/file-metadata-service/internal/domain"
)

var (
	ownerIDRE     = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)
	contentTypeRE = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9!#$&^_.+-]{0,126}/[a-zA-Z0-9][a-zA-Z0-9!#$&^_.+-]{0,126}$`)
	checksumRE    = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

func OwnerID(ownerID string) error {
	if !ownerIDRE.MatchString(ownerID) {
		return fmt.Errorf("%w: invalid owner_id format", domain.ErrInvalidInput)
	}
	if strings.Contains(ownerID, "..") {
		return fmt.Errorf("%w: invalid owner_id", domain.ErrInvalidInput)
	}
	return nil
}

func ContentType(contentType string) error {
	if !contentTypeRE.MatchString(contentType) {
		return fmt.Errorf("%w: invalid content_type", domain.ErrInvalidInput)
	}
	return nil
}

func ChecksumSHA256(checksum string) error {
	if checksum == "" {
		return nil
	}
	if !checksumRE.MatchString(strings.ToLower(checksum)) {
		return fmt.Errorf("%w: checksum_sha256 must be 64 hex chars", domain.ErrInvalidInput)
	}
	return nil
}

func FileSize(sizeBytes, maxSize int64) error {
	if sizeBytes <= 0 {
		return domain.ErrInvalidInput
	}
	if maxSize > 0 && sizeBytes > maxSize {
		return fmt.Errorf("%w: file exceeds max size", domain.ErrInvalidInput)
	}
	return nil
}

func CreateUploadInput(ownerID, originalName, contentType string, sizeBytes, maxSize int64) error {
	if originalName == "" || contentType == "" {
		return domain.ErrInvalidInput
	}
	if err := OwnerID(ownerID); err != nil {
		return err
	}
	if err := ContentType(contentType); err != nil {
		return err
	}
	return FileSize(sizeBytes, maxSize)
}
