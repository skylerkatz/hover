<?php

use Aws\Lambda\LambdaClient;
use GuzzleHttp\Promise\Utils;

class Warmer
{
    public LambdaClient $client;

    public function __construct()
    {
        $this->client = new LambdaClient([
            'region' => $_ENV['AWS_DEFAULT_REGION'],
            'version' => 'latest',
            'http' => [
                'timeout' => 5,
                'connect_timeout' => 5,
            ],
        ]);
    }

    public function warmContainers($containersToWarm): array
    {
        fwrite(STDERR, "Hover: Warming $containersToWarm containers.".PHP_EOL);

        $promises = collect(range(1, $containersToWarm - 1))
            ->mapWithKeys(function ($i) {
                return [
                    'warmer-'.$i => $this->client->invokeAsync([
                        'FunctionName' => $_ENV['AWS_LAMBDA_FUNCTION_NAME'],
                        'Qualifier' => $_ENV['AWS_LAMBDA_FUNCTION_VERSION'],
                        'LogType' => 'None',
                        'Payload' => json_encode(['warmer_ping' => true]),
                    ])
                ];
            })->all();

        try {
            Utils::settle($promises)->wait();
        } catch (\Throwable $e) {
            fwrite(STDERR, "Hover: Some warming invocations failed.".PHP_EOL);
        }

        fwrite(STDERR, "Hover: $containersToWarm containers have been warmed successfully.".PHP_EOL);

        return [
            'output' => 'Warming done!',
        ];
    }

    public function warmContainer(): array
    {
        usleep(50 * 1000);

        fwrite(STDERR, "Hover: 1 container warmed.".PHP_EOL);

        return [
            'output' => 'Warmed!',
        ];
    }
}