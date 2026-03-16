<?php


/*
//header ("Access-Control-Allow-Origin: *");
header ("Access-Control-Allow-Origin: http://127.0.0.1:4200");
header ("Access-Control-Expose-Headers: Content-Length, X-JSON");
header ("Access-Control-Allow-Credentials: true");
header ("Access-Control-Allow-Methods: GET, POST, PATCH, PUT, DELETE, OPTIONS");
header ("Access-Control-Allow-Headers: Origin, Content-Type,Accept,Authorization");
header ("Content-Type: application/json");

header ("Access-Control-Allow-Origin: http://localhost:4200");
header ("Access-Control-Allow-Headers: Origin, Content-Type,Accept,Authorization");
header ("Content-Type: application/json");



$AUTH_USER = 'admin';
$AUTH_PASS = '9f0d81b1742397d4c113f44e1366a98c';
header('Cache-Control: no-cache, must-revalidate, max-age=0');
$has_supplied_credentials = !(empty($_SERVER['PHP_AUTH_USER']) && empty($_SERVER['PHP_AUTH_PW']));
$is_not_authenticated = (
	!$has_supplied_credentials ||
	$_SERVER['PHP_AUTH_USER'] != $AUTH_USER ||
	$_SERVER['PHP_AUTH_PW']   != $AUTH_PASS
);
if ($is_not_authenticated) {
    echo 'not authorizedz';
    exit;
	header('HTTP/1.1 401 Authorization Required');
	header('WWW-Authenticate: Basic realm="Access denied"');
	exit;
}
*/
 
// header ("Access-Control-Allow-Origin: http://localhost:3000");



include_once('jwt_helper.php');

 
  $JWT_token = 'non_authorised';

$raw = file_get_contents('php://input');
$data = json_decode($raw,true);
//echo json_encode(array('username'=>$data['username']));
//echo json_encode(array('pass'=>$data['pass']));

 
	
?>