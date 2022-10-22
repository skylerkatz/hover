<?php

use hollodotme\FastCGI\Client;
use hollodotme\FastCGI\Exceptions\TimedoutException;
use hollodotme\FastCGI\Interfaces\ProvidesResponseData;
use hollodotme\FastCGI\SocketConnections\UnixDomainSocket;
use Symfony\Component\Process\Process;

class FpmManager
{
    const SOCKET_FILE = '/tmp/.hover/php-fpm.sock';
    const CONFIG_FILE = '/tmp/.hover/php-fpm.conf';
    const PID_FILE = '/tmp/.hover/php-fpm.pid';
    public Process $fpmProcess;
    public Client $fastCgiClient;
    public UnixDomainSocket $socketConnection;

    public function start()
    {
        if ($this->isReady()) {
            $this->killExistingFpm();
        }

        if (! is_dir(dirname(self::SOCKET_FILE))) {
            mkdir(dirname(self::SOCKET_FILE));
        }

        if (! file_exists(self::CONFIG_FILE)) {
            file_put_contents(
                self::CONFIG_FILE,
                file_get_contents(__DIR__.'/php-fpm.conf')
            );
        }

        $this->fpmProcess = new Process([
            'php-fpm',
            '--nodaemonize',
            '--force-stderr',
            '--fpm-config',
            self::CONFIG_FILE,
        ]);

        $this->fpmProcess->disableOutput()
            ->setTimeout(null)
            ->start(function ($type, $line) {
                fwrite(STDERR, $line.PHP_EOL);
            });

        $this->fastCgiClient = new Client();

        $this->socketConnection = new UnixDomainSocket(self::SOCKET_FILE, 1000, 900000);

        $this->waitUntilReady();
    }

    public function sendRequest(FpmRequest $request, $timeout): ProvidesResponseData
    {
        try {
            $socketId = $this->fastCgiClient->sendAsyncRequest($this->socketConnection, $request);

            $response = $this->fastCgiClient->readResponse($socketId, $timeout - 1000);
        } catch (TimedoutException $e) {
            $this->stop();

            $this->start();

            throw new Exception('FPM request timed out 1 second before Lambda times out.');
        }

        return $response;
    }

    protected function isReady()
    {
        clearstatcache(false, self::SOCKET_FILE);

        return file_exists(self::SOCKET_FILE);
    }

    private function killExistingFpm(): void
    {
        if (! file_exists(self::PID_FILE)) {
            unlink(self::SOCKET_FILE);

            return;
        }

        $pid = (int) file_get_contents(self::PID_FILE);

        if ($pid <= 0) {
            unlink(self::SOCKET_FILE);
            unlink(self::PID_FILE);

            return;
        }

        if (posix_getpgid($pid) === false) {
            unlink(self::SOCKET_FILE);
            unlink(self::PID_FILE);

            return;
        }

        if ($pid === posix_getpid()) {
            unlink(self::SOCKET_FILE);
            unlink(self::PID_FILE);

            return;
        }

        $result = posix_kill($pid, 15);

        if ($result === false) {
            unlink(self::SOCKET_FILE);
            unlink(self::PID_FILE);

            return;
        }

        $this->waitUntilStopped($pid);

        unlink(self::SOCKET_FILE);
        unlink(self::PID_FILE);
    }

    private function waitUntilStopped(int $pid): void
    {
        $wait = 5000;
        $timeout = 1000000;
        $elapsed = 0;

        while (posix_getpgid($pid) !== false) {
            usleep($wait);

            $elapsed += $wait;

            if ($elapsed > $timeout) {
                throw new Exception('Timeout while waiting for PHP-FPM to stop');
            }
        }
    }

    private function waitUntilReady(): void
    {
        $wait = 5000;
        $timeout = 5000000;
        $elapsed = 0;

        while (! $this->isReady()) {
            usleep($wait);

            $elapsed += $wait;

            if ($elapsed > $timeout) {
                throw new Exception('Timeout while waiting for PHP-FPM socket');
            }

            if (! $this->fpmProcess->isRunning()) {
                throw new Exception('PHP-FPM failed to start: '.PHP_EOL.$this->fpmProcess->getOutput().PHP_EOL.$this->fpmProcess->getErrorOutput());
            }
        }
    }

    public function stop(): void
    {
        if ($this->fpmProcess && $this->fpmProcess->isRunning()) {
            $this->fpmProcess->stop(0.5);

            if ($this->isReady()) {
                throw new Exception('PHP-FPM cannot be stopped');
            }
        }
    }

    public function __destruct()
    {
        $this->stop();
    }
}