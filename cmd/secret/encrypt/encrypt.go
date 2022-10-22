package encrypt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/aws/smithy-go/ptr"
	"github.com/spf13/cobra"
	"hover/aws"
	"hover/utils"
	"hover/utils/manifest"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type options struct {
	stage string
}

func Cmd() *cobra.Command {
	opts := options{}

	cmd := &cobra.Command{
		Use:   "encrypt --stage",
		Short: "Encrypt the secrets file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.stage == "" {
				return fmt.Errorf("you must specify a --stage")
			}

			return Run(&opts)
		},
	}

	cmd.Flags().StringVarP(&opts.stage, "stage", "s", "", "The stage name")

	return cmd
}

func Run(o *options) error {
	fmt.Println()

	plainFilePath := filepath.Join(utils.Path.Hover, o.stage+"-secrets.plain.env")
	encryptedFilePath := filepath.Join(utils.Path.Hover, o.stage+"-secrets.env")

	stage, err := manifest.Get(o.stage)
	if err != nil {
		return err
	}

	awsClient, _ := aws.New(stage.AwsProfile, stage.Region)

	plaintext, err := os.ReadFile(plainFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("plain secrets file doesn't exist at: " + plainFilePath)
		}
		return err
	}

	encryptionKey, err := getEncryptionKey(encryptedFilePath, stage, awsClient)
	if err != nil {
		return err
	}

	decoded, err := hex.DecodeString(encryptionKey)
	if err != nil {
		return err
	}

	secretKeyResults, err := awsClient.DecryptWithKms(stage.Name+"-secrets-key", decoded)
	if err != nil {
		return err
	}

	key := secretKeyResults.Plaintext

	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return err
	}

	plaintext = pad(plaintext, aes.BlockSize)

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	ciphertext := make([]byte, len(plaintext))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, plaintext)

	encryptedString := hex.EncodeToString(ciphertext)

	encryptedString += "------" + encryptionKey

	encryptedString += "------" + hex.EncodeToString(iv)

	err = os.WriteFile(encryptedFilePath, []byte(encryptedString), os.ModePerm)
	if err != nil {
		return err
	}

	err = os.Remove(plainFilePath)
	if err != nil {
		return err
	}

	utils.PrintInfo("Plain secrets file was deleted.")
	utils.PrintSuccess("Secrets file encrypted")

	return nil
}

func getEncryptionKey(encryptedFilePath string, stage *manifest.Manifest, awsClient *aws.Aws) (string, error) {
	_, err := os.Stat(encryptedFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			key, err := generateEncryptionKey(stage, awsClient)
			if err != nil {
				return "", err
			}
			return key, nil
		} else {
			return "", err
		}
	}

	encryptedFileContent, err := os.ReadFile(encryptedFilePath)
	if err != nil {
		return "", err
	}

	parts := strings.Split(string(encryptedFileContent), "------")
	if len(parts) < 3 {
		return "", fmt.Errorf("malformed file. It should contain three sections separated by six dashes. At:" + encryptedFilePath)
	}

	return parts[1], nil
}

func generateEncryptionKey(stage *manifest.Manifest, awsClient *aws.Aws) (string, error) {
	key := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", err
	}

	stringKey := hex.EncodeToString(key)

	kmsKeyName := ptr.String(stage.Name + "-secrets-key")

	// Ensure the KMS key used for encrypting the secrets encryption key exists.
	// If it doesn't exist, we create it.
	_, err := awsClient.GetKmsKey(kmsKeyName)
	if err != nil {
		if awsClient.KmsKeyDoesntExist(err) {
			err = awsClient.CreateKmsKey(kmsKeyName)
			if err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}

	result, err := awsClient.EncryptWithKms(*kmsKeyName, stringKey)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(result.CiphertextBlob), nil
}

func pad(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize

	padded := bytes.Repeat([]byte{byte(padding)}, padding)

	return append(ciphertext, padded...)
}
