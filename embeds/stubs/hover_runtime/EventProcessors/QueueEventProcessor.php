<?php

use Aws\Sqs\SqsClient;
use Illuminate\Contracts\Debug\ExceptionHandler;
use Illuminate\Queue\Jobs\SqsJob;
use Illuminate\Queue\WorkerOptions;
use Illuminate\Container\Container;

class QueueEventProcessor extends AbstractEventProcessor
{
    public Container $application;
    public Worker $worker;
    public static SqsJob $currentJob;
    public SqsClient $sqs;
    public array $config;

    public function __construct(Container $application, array $manifest)
    {
        $this->application = $application;

        [$isDownForMaintenance, $resetScope] = $this->extractWorkerDependencies();

        $this->worker = new Worker(
            $this->application['queue'],
            $this->application['events'],
            $this->application[ExceptionHandler::class],
            $isDownForMaintenance,
            $resetScope
        );

        $this->sqs = new SqsClient([
            'region' => $_ENV['AWS_DEFAULT_REGION'],
            'version' => 'latest',
            'http' => [
                'timeout' => 60,
                'connect_timeout' => 60,
            ],
        ]);

        $queueName = substr(
            str_replace($manifest['name'].'-', '', $_ENV['AWS_LAMBDA_FUNCTION_NAME']),
            0, -6
        );

        $this->config = $manifest['queue'][$queueName];
    }

    public function process(array $invocationBody, string $invocationId, int $invocationDeadline): array
    {
        $timeout = $invocationDeadline - intval(microtime(true) * 1000);

        $jobData = [
            'MessageId' => $invocationBody['Records'][0]['messageId'],
            'ReceiptHandle' => $invocationBody['Records'][0]['receiptHandle'],
            'Body' => $invocationBody['Records'][0]['body'],
            'Attributes' => $invocationBody['Records'][0]['attributes'],
            'MessageAttributes' => $invocationBody['Records'][0]['messageAttributes'],
        ];

        $queueUrl = $this->getQueueUrl($invocationBody['Records'][0]);

        self::$currentJob = new SqsJob(
            $this->application,
            $this->sqs,
            $jobData,
            'sqs',
            $queueUrl
        );

        $workerOptions = new WorkerOptions();

        $workerOptions->sleep = 0;
        $workerOptions->maxJobs = 1;
        $workerOptions->timeout = ceil($timeout / 1000) - 1;
        $workerOptions->maxTries = $this->config['tries'] ?? 1;
        $workerOptions->backoff = $this->config['backoff'] ?? 0;

        $this->worker->daemon('sqs', $queueUrl, $workerOptions);

        if (self::$currentJob->isReleased()) {
            return [
                "batchItemFailures" => [
                    ['itemIdentifier' => $invocationBody['Records'][0]['messageId']]
                ]
            ];
        }

        return [];
    }

    protected function extractWorkerDependencies(): array
    {
        $worker = $this->application->make('queue.worker');

        $workerReflection = new ReflectionClass($worker);

        $isDownForMaintenanceProperty = $workerReflection->getProperty('isDownForMaintenance');
        $isDownForMaintenanceProperty->setAccessible(true);

        $resetScopeProperty = $workerReflection->getProperty('resetScope');
        $resetScopeProperty->setAccessible(true);

        return [
            $isDownForMaintenanceProperty->getValue($worker),
            $resetScopeProperty->getValue($worker)
        ];
    }

    protected function getQueueUrl(array $message)
    {
        $eventSourceArn = explode(':', $message['eventSourceARN']);

        return sprintf(
            'https://sqs.%s.amazonaws.com/%s/%s',
            $message['awsRegion'],
            $eventSourceArn[4],
            $eventSourceArn[5]
        );
    }
}