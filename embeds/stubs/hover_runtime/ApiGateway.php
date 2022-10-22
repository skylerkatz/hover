<?php

use hollodotme\FastCGI\Interfaces\ProvidesResponseData;
use Illuminate\Support\Str;

class ApiGateway
{
    public function transformResponse(ProvidesResponseData $response): array
    {
        [$headers, $cookies] = $this->extractHeadersAndCookies($response->getHeaders());
        $requiresEncoding = $this->isBase64EncodingRequired($headers);

        return [
            'cookies' => $cookies,
            'isBase64Encoded' => $requiresEncoding,
            'statusCode' => $this->getStatusFromHeaders($headers),
            'headers' => $headers,
            'body' => $requiresEncoding
                ? base64_encode($response->getBody())
                : $response->getBody(),
        ];
    }

    private function extractHeadersAndCookies(array $incomingHeaders): array
    {
        $headers = [];
        $cookies = [];

        foreach ($incomingHeaders as $name => $values) {
            $name = str_replace(' ', '-', ucwords(str_replace('-', ' ', $name)));

            if ($name === 'Set-Cookie') {
                $cookies = is_array($values) ? $values : [$values];
            } else {
                $headers[$name] = is_array($values) ? end($values) : $values;
            }
        }

        return [$headers, $cookies];
    }

    private function getStatusFromHeaders(array $headers): int
    {
        $headers = array_change_key_case($headers, CASE_LOWER);

        return isset($headers['status'])
            ? (int) explode(' ', $headers['status'])[0]
            : 200;
    }

    public static function isBase64EncodingRequired($headers): bool
    {
        $contentType = $headers['Content-Type'] ?? 'text/html';

        if (Str::startsWith($contentType, 'text/') ||
            Str::contains($contentType, ['xml', 'json'])) {
            return false;
        }

        return true;
    }
}