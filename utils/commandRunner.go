package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hover/aws"
	"strings"
)

func RunCommand(lambdaName *string, version *string, commandString string, aws *aws.Aws) (string, any, error) {
	PrintInfo("Executing 'php artisan " + strings.TrimSpace(commandString) + "'")

	result, err := aws.InvokeLambda(lambdaName, version, []byte("{\"command\": \""+commandString+"\"}"))
	if err != nil {
		return "", nil, err
	}

	var output map[string]any

	err = json.Unmarshal(result.Payload, &output)
	if err != nil {
		return "", nil, err
	}

	if output["Error"] != nil {
		return "", nil, fmt.Errorf(output["Error"].(string))
	}

	if output["output"] == nil {
		return "", nil, fmt.Errorf("failed to execute command")
	}

	outputString, err := base64.StdEncoding.DecodeString(output["output"].(string))
	if err != nil {
		return "", nil, err
	}

	return string(outputString), output["exit_code"], nil
}
