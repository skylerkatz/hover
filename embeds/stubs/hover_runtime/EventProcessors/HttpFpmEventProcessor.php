<?php

class HttpFpmEventProcessor extends AbstractEventProcessor
{
    private ApiGateway $apiGateway;
    private FpmManager $fpmManager;
    private Warmer $warmer;
    private $count;

    public function __construct(ApiGateway $apiGateway, FpmManager $fpmManager, Warmer $warmer)
    {
        $this->apiGateway = $apiGateway;
        $this->fpmManager = $fpmManager;
        $this->warmer = $warmer;
    }

    public function process(array $invocationBody, string $invocationId, int $invocationDeadline): array
    {
        $this->count++;

        if (isset($invocationBody['warmer'])) {
            return $this->warmer->warmContainers($invocationBody['containers']);
        }

        if (isset($invocationBody['warmer_ping'])) {
            if ($this->count == 1) {
                $this->sendRequestToFpm(
                    $invocationBody, $invocationId, $invocationDeadline
                );
            }

            return $this->warmer->warmContainer();
        }

        if (! isset($invocationBody['requestContext'])) {
            throw new Exception('Unexpected invocation type!');
        }

        $response = $this->sendRequestToFpm(
            $invocationBody, $invocationId, $invocationDeadline
        );

        return $this->apiGateway->transformResponse($response);
    }

    public function sendRequestToFpm($invocationBody, $invocationId, $invocationDeadline)
    {
        return $this->fpmManager->sendRequest(
            new FpmRequest($invocationBody, [
                'AWS_REQUEST_ID' => $invocationId,
                'AWS_REQUEST_DEADLINE' => $invocationDeadline,
            ]),
            $invocationDeadline - intval(microtime(true) * 1000)
        );
    }
}