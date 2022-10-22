<?php

use hollodotme\FastCGI\Interfaces\ProvidesRequestData;

class FpmRequest implements ProvidesRequestData
{
    public array $event;
    public array $headers;
    public string $requestBody;
    public array $serverVariables;

    public function __construct(array $event, $serverVariables = [])
    {
        $this->event = $event;
        $this->headers = $this->getHeaders($event);
        $this->requestBody = $this->getRequestBody($event);
        $this->serverVariables = $serverVariables;
    }

    public function getGatewayInterface(): string
    {
        return 'FastCGI/1.0';
    }

    public function getRequestMethod(): string
    {
        return $this->event['httpMethod'] ?? $this->event['requestContext']['http']['method'] ?? 'GET';
    }

    public function getScriptFilename(): string
    {
        return __DIR__.'/index.php';
    }

    public function getServerSoftware(): string
    {
        return 'hover';
    }

    public function getRemoteAddress(): string
    {
        return '127.0.0.1';
    }

    public function getRemotePort(): int
    {
        return $this->headers['x-forwarded-port'] ?? 80;
    }

    public function getServerAddress(): string
    {
        return '127.0.0.1';
    }

    public function getServerPort(): int
    {
        return $this->headers['x-forwarded-port'] ?? 80;
    }

    public function getServerName(): string
    {
        return $this->headers['host'] ?? 'localhost';
    }

    public function getServerProtocol(): string
    {
        return $this->event['requestContext']['http']['protocol'] ?? 'HTTP/1.1';
    }

    public function getContentType(): string
    {
        return $this->headers['content-type'] ?? '';
    }

    public function getContentLength(): int
    {
        return strlen($this->requestBody);
    }

    public function getContent(): string
    {
        return $this->requestBody;
    }

    public function getCustomVars(): array
    {
        return [];
    }

    public function getParams(): array
    {
        foreach ($this->headers as $header => $value) {
            $this->serverVariables['HTTP_'.strtoupper(str_replace('-', '_', $header))] = $value;
        }

        $queryString = $this->getQueryString();
        $uri = $this->getRequestUri();

        return array_merge($this->serverVariables, [
            'GATEWAY_INTERFACE' => $this->getGatewayInterface(),
            'REQUEST_METHOD' => $this->getRequestMethod(),
            'REQUEST_URI' => empty($queryString) ? $uri : $uri.'?'.$queryString,
            'SCRIPT_FILENAME' => $this->getScriptFilename(),
            'SERVER_SOFTWARE' => $this->getServerSoftware(),
            'REMOTE_ADDR' => $this->getRemoteAddress(),
            'REMOTE_PORT' => $this->getRemotePort(),
            'SERVER_ADDR' => $this->getServerAddress(),
            'SERVER_PORT' => $this->getServerPort(),
            'SERVER_NAME' => $this->getServerName(),
            'SERVER_PROTOCOL' => $this->getServerProtocol(),
            'CONTENT_TYPE' => $this->getContentType(),
            'CONTENT_LENGTH' => $this->getContentLength(),
            'PATH_INFO' => $uri,
            'QUERY_STRING' => $queryString,
        ]);
    }

    public function getRequestUri(): string
    {
        return $this->event['rawPath'] ?? '/hover-dummy-route';
    }

    public function getResponseCallbacks(): array
    {
        return [];
    }

    public function getFailureCallbacks(): array
    {
        return [];
    }

    public function getPassThroughCallbacks(): array
    {
        return [];
    }

    protected function getHeaders(array $event): array
    {
        $headers = array_change_key_case($event['headers'] ?? [], CASE_LOWER);

        if (isset($event['cookies']) && ! empty($event['cookies'])) {
            $headers['cookie'] = implode('; ', $event['cookies']);
        }

        return $headers;
    }

    protected function getRequestBody(array $event)
    {
        return isset($event['isBase64Encoded']) && $event['isBase64Encoded']
            ? base64_decode($event['body'] ?? '')
            : $event['body'] ?? '';
    }

    protected function getQueryString()
    {
        $queryString = '';

        if (isset($this->event['rawQueryString'])) {
            parse_str($this->event['rawQueryString'], $params);

            $queryString = http_build_query($params);
        }

        return $queryString;
    }
}