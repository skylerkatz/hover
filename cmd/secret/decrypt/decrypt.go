package decrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
	"github.com/spf13/cobra"
	"hover/aws"
	"hover/utils"
	"hover/utils/manifest"
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
		Use:   "decrypt --stage",
		Short: "Decrypt the secrets file",
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

	err = ensurePLainSecretsFileIsGitIgnored()
	if err != nil {
		return err
	}

	secrets, encryptionKey, iv, err := extractParameters(encryptedFilePath)
	if err != nil {
		return err
	}

	secretKeyResults, err := awsClient.DecryptWithKms(stage.Name+"-secrets-key", encryptionKey)
	if err != nil {
		return err
	}

	block, err := aes.NewCipher(secretKeyResults.Plaintext)
	if err != nil {
		panic(err)
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(secrets, secrets)

	secrets = unPad(secrets)

	err = os.WriteFile(plainFilePath, secrets, os.ModePerm)
	if err != nil {
		return err
	}

	utils.PrintInfo("Plain secrets file was added to `.hover/.gitignore`.")
	utils.PrintSuccess("Secrets file decrypted")

	return nil
}

func extractParameters(encryptedFilePath string) ([]byte, []byte, []byte, error) {
	_, err := os.Stat(encryptedFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil, fmt.Errorf("encrypted secrets file doesn't exist at: " + encryptedFilePath)
		} else {
			return nil, nil, nil, err
		}
	}

	encryptedFileContent, err := os.ReadFile(encryptedFilePath)
	if err != nil {
		return nil, nil, nil, err
	}

	parts := strings.Split(string(encryptedFileContent), "------")

	if len(parts) < 3 {
		return nil, nil, nil, fmt.Errorf("encrypted secrets file is malformed")
	}

	secrets, err := hex.DecodeString(parts[0])
	if err != nil {
		return nil, nil, nil, err
	}

	key, err := hex.DecodeString(parts[1])
	if err != nil {
		return nil, nil, nil, err
	}

	iv, err := hex.DecodeString(parts[2])
	if err != nil {
		return nil, nil, nil, err
	}

	return secrets, key, iv, err
}

func ensurePLainSecretsFileIsGitIgnored() error {
	path := filepath.Join(utils.Path.Hover, ".gitignore")

	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("a .gitignore file doesn't exist inside the .hover directory. Create one first")
		}
		return err
	}

	gitIgnoreContent, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if !strings.Contains(string(gitIgnoreContent), "*-secrets.plain.env") {
		err = os.WriteFile(path, []byte(string(gitIgnoreContent)+"\n*-secrets.plain.env"), os.ModePerm)
		if err != nil {
			return err
		}
	}

	return nil
}

func unPad(content []byte) []byte {
	padding := len(content)

	unPadded := int(content[padding-1])

	return content[:(padding - unPadded)]
}
