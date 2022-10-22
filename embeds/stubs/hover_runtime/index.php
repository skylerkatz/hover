<?php

use Illuminate\Contracts\Http\Kernel as HttpKernelContract;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Facade;

define('LARAVEL_START', microtime(true));

require __DIR__.'/../vendor/autoload.php';

$app = require_once __DIR__.'/../bootstrap/app.php';

$app->useStoragePath('/tmp/storage');

$kernel = $app->make(HttpKernelContract::class);

$app->instance('request', $request = Request::capture());

Facade::clearResolvedInstance('request');

$kernel->bootstrap();

$response = $kernel->handle($request);

$kernel->terminate($request, $response);

$response->send();
