<?php

abstract class AbstractEventProcessor
{
    public abstract function process(array $invocationBody, string $invocationId, int $invocationDeadline): array;
}