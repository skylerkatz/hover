package build

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"hover/embeds"
	"hover/utils"
	"hover/utils/manifest"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type options struct {
	alias     string
	skipTests bool
}

func Cmd() *cobra.Command {
	opts := options{}

	cmd := &cobra.Command{
		Use:   "build <ALIAS>",
		Args:  cobra.ExactArgs(1),
		Short: "Create a new build for the defined stage",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.alias = args[0]

			return Run(&opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.skipTests, "skip-tests", "", false, "Skip running tests")

	return cmd
}

func Run(o *options) error {
	fmt.Println()

	fmt.Print(heredoc.Doc(`
		██╗  ██╗ ██████╗ ██╗   ██╗███████╗██████╗ 
		██║  ██║██╔═══██╗██║   ██║██╔════╝██╔══██╗
		███████║██║   ██║██║   ██║█████╗  ██████╔╝
		██╔══██║██║   ██║╚██╗ ██╔╝██╔══╝  ██╔══██╗
		██║  ██║╚██████╔╝ ╚████╔╝ ███████╗██║  ██║
		╚═╝  ╚═╝ ╚═════╝   ╚═══╝  ╚══════╝╚═╝  ╚═╝
`))

	fmt.Println()

	stage, err := manifest.Get(o.alias)
	if err != nil {
		return err
	}

	err = ensurePlainSecretsFileDoesntExist(o.alias)
	if err != nil {
		return err
	}

	err = deleteOutDirectory()
	if err != nil {
		return err
	}

	addRuntime(stage, o.alias)
	addManifest(*stage)

	err = runDockerBuild(stage, o)
	if err != nil {
		return err
	}

	return nil
}

func ensurePlainSecretsFileDoesntExist(alias string) error {
	filename := alias + "-secrets.plain.env"

	_, err := os.Stat(filepath.Join(utils.Path.Hover, filename))
	if err == nil {
		return fmt.Errorf(heredoc.Doc(`
			Cannot build the stage while the plain secrets file ".hover/` + filename + `" is present. 
			Make sure to delete it or run "hover secret encrypt --stage=` + alias + `" to encrypt the latest content.
`))
	}

	return nil
}

func deleteOutDirectory() error {
	err := os.RemoveAll(utils.Path.Out)
	if err != nil {
		return err
	}

	return nil
}

func addRuntime(stage *manifest.Manifest, alias string) {
	fs.WalkDir(embeds.HoverRuntimeStubs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path == "." {
			return nil
		}

		if path == "stubs" {
			return nil
		}

		var newPath = strings.Replace(path, "stubs", "", -1)
		newPath = filepath.Join(utils.Path.ApplicationOut, newPath)

		if d.IsDir() {
			os.MkdirAll(newPath, os.ModePerm)

			return nil
		}

		content, _ := embeds.HoverRuntimeStubs.ReadFile(path)

		os.WriteFile(newPath, content, os.ModePerm)

		return nil
	})

	encryptedSecretsFilePath := filepath.Join(utils.Path.Hover, alias+"-secrets.env")

	// TODO handle errors
	_, err := os.Stat(encryptedSecretsFilePath)
	if err != nil {
		fmt.Println(err.Error())
		// Do nothing
	} else {
		encryptedSecretsFileContent, err := os.ReadFile(encryptedSecretsFilePath)
		if err != nil {
			//return err
		}

		os.WriteFile(filepath.Join(utils.Path.ApplicationOut, "hover_runtime", ".env"), encryptedSecretsFileContent, os.ModePerm)
	}
}

func addManifest(stage manifest.Manifest) {
	buildId := uuid.NewString()

	manifestJson, _ := json.MarshalIndent(stage, "", "\t")
	hash := md5.Sum(manifestJson)

	stage.BuildDetails.Id = buildId
	stage.BuildDetails.Hash = hex.EncodeToString(hash[:])
	stage.BuildDetails.Time = time.Now().Unix()

	jsonContent, _ := json.MarshalIndent(stage, "", "\t")

	os.WriteFile(filepath.Join(utils.Path.ApplicationOut, "hover_runtime", "manifest.json"), jsonContent, os.ModePerm)
}

func runDockerBuild(stage *manifest.Manifest, o *options) error {
	utils.PrintStep("Building the base container image")

	dockerFilePath := filepath.Join(utils.Path.Hover, stage.Dockerfile)

	dockerFile, err := os.ReadFile(dockerFilePath)
	if err != nil {
		return err
	}

	dockerFileContent := string(dockerFile)

	err = utils.Exec(fmt.Sprintf("docker build --file=%s --tag=%s .",
		dockerFilePath,
		stage.Name+":latest",
	), utils.Path.Current)
	if err != nil {
		return err
	}

	if !o.skipTests && strings.Contains(dockerFileContent, "FROM base as tests") {
		utils.PrintStep("Building the tests container image")

		err = utils.Exec(fmt.Sprintf("docker build --target=tests --file=%s --tag=%s .",
			dockerFilePath,
			stage.Name+":latest-tests",
		), utils.Path.Current)
		if err != nil {
			return err
		}

		utils.PrintStep("Running tests on the tests image")

		err = utils.Exec(fmt.Sprintf("docker run --rm %s",
			stage.Name+":latest-tests",
		), utils.Path.Current)
		if err != nil {
			return err
		}
	}

	utils.PrintStep("Building the assets container image")

	err = utils.Exec(fmt.Sprintf("docker build --target=assets --file=%s --tag=%s .",
		dockerFilePath,
		stage.Name+":latest-assets",
	), utils.Path.Current)
	if err != nil {
		return err
	}

	utils.PrintStep("Moving assets to .hover/out/assets")

	err = utils.Exec(fmt.Sprintf("docker run --rm -v /%s/assets:/out %s cp -R public/. /out",
		utils.Path.Out,
		stage.Name+":latest-assets",
	), utils.Path.Current)
	if err != nil {
		return err
	}

	return nil
}
