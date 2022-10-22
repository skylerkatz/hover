<?php

use Aws\Kms\KmsClient;
use Dotenv\Dotenv;
use Illuminate\Contracts\Console\Kernel as ConsoleKernelContract;

class Hover
{
    public array $manifest;

    public function __construct(array $manifest)
    {
        $this->manifest = $manifest;
    }

    public function createStorageDirectories()
    {
        $directories = [
            '/tmp/storage/app',
            '/tmp/storage/logs',
            '/tmp/storage/bootstrap/cache',
            '/tmp/storage/framework/cache',
            '/tmp/storage/framework/views',
        ];

        foreach ($directories as $directory) {
            if (! is_dir($directory)) {
                mkdir($directory, 0755, true);
            }
        }
    }

    public function getAppInstance(string $appRoot)
    {
        $app = require $appRoot.'/bootstrap/app.php';

        $app->useStoragePath('/tmp/storage');;

        return $app;
    }

    public function cacheLaravelStuff($app)
    {
        $app->make(ConsoleKernelContract::class)->call('config:cache');
    }

    public function populateEnvironmentVariables()
    {
        fwrite(STDERR, "Hover: populating stage variables.".PHP_EOL);

        $values = array_merge([
            'ASSET_URL' => 'https://'.$_ENV['CF_DOMAIN'].'/'.$this->manifest['build_details']['id']
        ], $this->manifest['environment']);

        foreach ($values as $key => $value) {
            $_ENV[$key] = $value;
            $_SERVER[$key] = $value;
        }

        $secretsPath = "/var/task/hover_runtime/.env";

        if (file_exists($secretsPath)) {
            $client = new KmsClient([
                'region' => $_ENV['AWS_DEFAULT_REGION'],
                'version' => 'latest',
            ]);

            $encryptedSecrets = file_get_contents($secretsPath);

            [$content, $key, $iv] = explode('------', $encryptedSecrets);

            fwrite(STDERR, "Hover: populating stage secrets.".PHP_EOL);

            $encryptionKeyResponse = $client->decrypt([
                'KeyId' => "alias/{$this->manifest['name']}-secrets-key",
                "CiphertextBlob" => hex2bin($key)
            ]);

            $encryptionKey = $encryptionKeyResponse['Plaintext'];

            $decryptedSecrets = \openssl_decrypt(
                hex2bin($content), 'aes-256-cbc', $encryptionKey, true, hex2bin($iv)
            );

            file_put_contents("/tmp/.env.hover", $decryptedSecrets);

            $dotenv = Dotenv::createImmutable("/tmp", ".env.hover");

            $dotenv->load();
        }
    }
}