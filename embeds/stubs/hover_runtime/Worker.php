<?php

use Illuminate\Queue\Worker as LaravelWorker;
use Illuminate\Queue\WorkerOptions;

class Worker extends LaravelWorker
{
    protected function daemonShouldRun(WorkerOptions $options, $connectionName, $queue)
    {
        return true;
    }

    public function kill($status = 0)
    {
        throw new Exception('Job timed out. It will be retried again.');
    }

    protected function getNextJob($connection, $queue)
    {
        return QueueEventProcessor::$currentJob;
    }

    protected function timeoutForJob($job, WorkerOptions $options)
    {
        return min(parent::timeoutForJob($job, $options), $options->timeout);
    }

    protected function stopIfNecessary(WorkerOptions $options, $lastRestart, $startTime = 0, $jobsProcessed = 0, $job = null)
    {
        return $options->maxJobs && $jobsProcessed >= $options->maxJobs ? 0 : null;
    }
}