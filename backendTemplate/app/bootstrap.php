<?php
// Load environment variables first
require_once __DIR__ . '/helpers/Env.php';
Env::load(__DIR__ . '/.env');

//Load Config
require_once __DIR__ . '/config/config.php';
require_once APPROOT . '/config/email_config.php';

// Load helpers
require_once APPROOT . '/helpers/jwt_helper.php';
require_once APPROOT . '/helpers/QRCode_helper.php';
require_once APPROOT . '/helpers/headers.php';
require_once APPROOT . '/helpers/Exception.php';
require_once APPROOT . '/helpers/PHPMailer.php';
require_once APPROOT . '/helpers/SMTP.php';
require_once APPROOT . '/helpers/phpMailer/PHPMailerAutoload.php';

//Autoload Core Libraries
spl_autoload_register(function ($className) {
    $paths = [
        APPROOT . '/libraries/',
        APPROOT . '/models/',
        APPROOT . '/controllers/',
        APPROOT . '/services/'
    ];

    foreach ($paths as $path) {
        $file = $path . $className . '.php';
        if (file_exists($file)) {
            require_once $file;
            return;
        }
    }
});
