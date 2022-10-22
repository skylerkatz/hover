<?php

class Lambda
{
    public $curlNextInvocationHandle;
    public $curlLambdaResponseHandle;
    public $runtimeApi = '';

    public function __construct()
    {
        $this->runtimeApi = getenv('AWS_LAMBDA_RUNTIME_API');
    }

    public function getNextInvocation(): array
    {
        $invocationId = '';
        $invocationDeadline = 0;
        $invocationBody = '';

        if ($this->curlNextInvocationHandle === null) {
            $this->curlNextInvocationHandle = curl_init("http://{$this->runtimeApi}/2018-06-01/runtime/invocation/next");

            curl_setopt($this->curlNextInvocationHandle, CURLOPT_FOLLOWLOCATION, true);
            curl_setopt($this->curlNextInvocationHandle, CURLOPT_FAILONERROR, true);
        }

        curl_setopt($this->curlNextInvocationHandle, CURLOPT_HEADERFUNCTION, function ($ch, $header) use (&$invocationId, &$invocationDeadline) {
            if (! preg_match('/:\s*/', $header)) {
                return strlen($header);
            }

            [$name, $value] = preg_split('/:\s*/', $header, 2);

            if (strtolower($name) === 'lambda-runtime-aws-request-id') {
                $invocationId = trim($value);
            }

            if (strtolower($name) === 'lambda-runtime-deadline-ms') {
                $invocationDeadline = intval($value);
            }

            return strlen($header);
        });

        curl_setopt($this->curlNextInvocationHandle, CURLOPT_WRITEFUNCTION, function ($ch, $chunk) use (&$invocationBody) {
            $invocationBody .= $chunk;

            return strlen($chunk);
        });

        curl_exec($this->curlNextInvocationHandle);

        if (curl_error($this->curlNextInvocationHandle)) {
            $message = curl_error($this->curlNextInvocationHandle);

            $this->closeCurlHandleForNextInvocation();

            throw new Exception('Failed to get the lambda invocation: '.$message);
        }

        if ($invocationId === '') {
            throw new Exception('Failed to get the lambda invocation id');
        }

        if ($invocationBody === '') {
            throw new Exception('Failed to get the lambda invocation body');
        }

        return [json_decode($invocationBody, true), $invocationId, $invocationDeadline];
    }

    public function processInvocation(array $invocationBody, string $invocationId, int $invocationDeadline, AbstractEventProcessor $processor): void
    {
        try {
            $result = $processor->process($invocationBody, $invocationId, $invocationDeadline);
        } catch (\Throwable $e) {
            $error = [
                'errorMessage' => $e->getMessage(),
                'errorType' => get_class($e),
                'stackTrace' => explode(PHP_EOL, $e->getTraceAsString()),
            ];

            fwrite(STDERR, "Hover: Something went wrong => ".$e->getMessage().PHP_EOL);

            echo json_encode($e->getTrace());

            $this->sendFailureResponseToLambda($invocationId, $error);

            return;
        }

        $this->sendSuccessResponseToLambda($invocationId, $result);
    }

    private function sendSuccessResponseToLambda($invocationId, $responseBody)
    {
        $this->respondToLambda(
            "http://{$this->runtimeApi}/2018-06-01/runtime/invocation/{$invocationId}/response",
            $responseBody
        );
    }

    private function sendFailureResponseToLambda($invocationId, array $error)
    {
        $this->respondToLambda(
            "http://{$this->runtimeApi}/2018-06-01/runtime/invocation/{$invocationId}/error",
            $error
        );
    }

    public function sendInitializationFailureResponseToLambda(Throwable $exception)
    {
        $this->respondToLambda(
            "http://{$this->runtimeApi}/2018-06-01/runtime/init/error",
            [
                'errorMessage' => $exception->getMessage(),
                'errorType' => get_class($exception),
                'stackTrace' => explode(PHP_EOL, $exception->getTraceAsString()),
            ]
        );
    }

    private function respondToLambda($url, $data)
    {
        $json = json_encode($data);

        if ($json === false) {
            throw new Exception('Error encoding the Laravel response into JSON. Seems like you are responding with a file: '.json_last_error_msg());
        }

        if ($this->curlLambdaResponseHandle === null) {
            $this->curlLambdaResponseHandle = curl_init();

            curl_setopt($this->curlLambdaResponseHandle, CURLOPT_CUSTOMREQUEST, 'POST');
            curl_setopt($this->curlLambdaResponseHandle, CURLOPT_RETURNTRANSFER, true);
            curl_setopt($this->curlLambdaResponseHandle, CURLOPT_FAILONERROR, true);
        }

        curl_setopt($this->curlLambdaResponseHandle, CURLOPT_URL, $url);
        curl_setopt($this->curlLambdaResponseHandle, CURLOPT_POSTFIELDS, $json);
        curl_setopt($this->curlLambdaResponseHandle, CURLOPT_HTTPHEADER, [
            'Content-Type: application/json',
            'Content-Length: '.strlen($json),
        ]);

        curl_exec($this->curlLambdaResponseHandle);

        if (curl_error($this->curlLambdaResponseHandle)) {
            $message = curl_error($this->curlLambdaResponseHandle);

            $this->closeCurlHandleForLambdaResponse();

            throw new Exception('Error calling the runtime API: '.$message);
        }
    }

    private function closeCurlHandleForNextInvocation(): void
    {
        if ($this->curlNextInvocationHandle !== null) {
            curl_close($this->curlNextInvocationHandle);

            $this->curlNextInvocationHandle = null;
        }
    }

    private function closeCurlHandleForLambdaResponse(): void
    {
        if ($this->curlLambdaResponseHandle !== null) {
            curl_close($this->curlLambdaResponseHandle);

            $this->curlLambdaResponseHandle = null;
        }
    }
}