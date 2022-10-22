rm -rf  public/* storage/framework/* storage/logs/* storage/cache/* storage/debugbar/* database/factories/* \
        resources/css/* resources/js/* bootstrap/cache/routes*.php bootstrap/cache/config.php database/*.sqlite \
        package-lock.json tests/* /root/.composer /usr/local/bin/composer /tmp/pear /var/cache/apk/*

mv /var/task/.hover/out/application/hover_runtime/bootstrap /opt/bootstrap
cp -R /var/task/.hover/out/application/hover_runtime /var/task/hover_runtime

rm -rf .hover
rm /var/task/.env

chmod 755 /opt/bootstrap

sed -i -- 's/AWS_ACCESS_KEY_ID/NULL_AWS_ACCESS_KEY_ID/g' /var/task/config/*.php
sed -i -- 's/AWS_SESSION_TOKEN/NULL_AWS_SESSION_TOKEN/g' /var/task/config/*.php
sed -i -- 's/AWS_SECRET_ACCESS_KEY/NULL_AWS_SECRET_ACCESS_KEY/g' /var/task/config/*.php
sed -i -- 's/<?php/<?php \n ini_set("display_errors", "1"); \n error_reporting(E_ALL);/g' /var/task/artisan
sed -i -- 's/^\$app =\(.*\)/\$app =\1 \n$app->useStoragePath("\/tmp\/storage");/g' /var/task/artisan