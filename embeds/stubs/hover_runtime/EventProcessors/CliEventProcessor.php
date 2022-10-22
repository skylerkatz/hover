<?php

use Symfony\Component\Process\Exception\ProcessTimedOutException;
use Symfony\Component\Process\Process;

class CliEventProcessor extends AbstractEventProcessor
{
    public function process(array $invocationBody, string $invocationId, int $invocationDeadline): array
    {
        $timeout = $invocationDeadline - intval(microtime(true) * 1000);

        fwrite(STDERR,
            sprintf('Hover: Executing php artisan %s', trim($invocationBody['command'])).PHP_EOL
        );

        $process = Process::fromShellCommandline(
            sprintf('php %s/artisan %s --no-interaction 2>&1',
                $_ENV['LAMBDA_TASK_ROOT'],
                trim($invocationBody['command'])
            )
        )->setTimeout(ceil($timeout / 1000) - 1);

        try {
            $process->run(function ($type, $line) use (&$output) {
                fwrite(STDERR, $line.PHP_EOL);

                $output .= $line;
            });
        } catch (ProcessTimedOutException $e) {
            throw new Exception('CLI command timed out 1 second before Lambda times out.');
        }

        return [
            'exit_code' => $process->getExitCode(),
            'output' => base64_encode($output),
        ];
    }
}