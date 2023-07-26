<?php

define('__ROOT__', dirname(dirname(__FILE__)));
require_once(__ROOT__.'/Persistent.php');
require_once(__ROOT__.'/WebAuthn/WebAuthn.php');
@persistent_start();

try {
  // route parse
  $fn = '';
  $url = parse_url($_SERVER['REQUEST_URI']);

  switch ($url['path']) {
    case '/webauthn/getGetArgs':
    case '/webauthn/getCreateArgs':
    case '/webauthn/processGet':
    case '/webauthn/processCreate':
    case '/webauthn/clearRegistrations':
      $fn = substr($url['path'], strlen('/webauthn/'));
      break;
    case '/webauthn/webauthn.js':
      header('Content-Type: application/javascript');
      readfile('webauthn.js');
      exit();
      break;
    case '/':
    case '/index.html':
      readfile('index.html');
      exit();
      break;
    default:
      http_response_code(404);
      die();
  }

  // read get argument and post body
  $userId = filter_input(INPUT_GET, 'userId', FILTER_SANITIZE_SPECIAL_CHARS);
  $userName = filter_input(INPUT_GET, 'userName', FILTER_SANITIZE_SPECIAL_CHARS);
  $userDisplayName = filter_input(INPUT_GET, 'userDisplayName', FILTER_SANITIZE_SPECIAL_CHARS);

  if ($userId) {
    $userId = preg_replace('/[^0-9a-f]/i', '', $userId);
  }
  if ($userName) {
    $userName = preg_replace('/[^0-9a-z]/i', '', $userName);
  }
  if ($userDisplayName) {
    $userDisplayName = preg_replace('/[^0-9a-z]/i', '', $userDisplayName);
  }

  $post = trim(file_get_contents('php://input'));
  if ($post) {
    $post = json_decode($post, null, 512, JSON_THROW_ON_ERROR);
  }

  // Formats
  $formats = [];
  $formats[] = 'android-key';
  $formats[] = 'android-safetynet';
  $formats[] = 'apple';
  $formats[] = 'fido-u2f';
  $formats[] = 'none';
  $formats[] = 'packed';
  $formats[] = 'tpm';

  // cross-platform: true, if type internal is not allowed
  //                 false, if only internal is allowed
  //                 null, if internal and cross-platform is allowed
  $crossPlatformAttachment = true;

  // new Instance of the server library.
  // make sure that $rpId is the domain name.
  $WebAuthn = new lbuchs\WebAuthn\WebAuthn(getenv('WEBAUTHN_RP_NAME'), getenv('WEBAUTHN_RP_ID'), $formats);

  if ($fn === 'getCreateArgs') {
    // ------------------------------------
    // request for create arguments
    // ------------------------------------

    $createArgs = $WebAuthn->getCreateArgs(\hex2bin($userId), $userName, $userDisplayName, 20, 'discouraged', 'discouraged', $crossPlatformAttachment);

    header('Content-Type: application/json');
    print(json_encode($createArgs));

    // save challange to session. you have to deliver it to processGet later.
    $_SESSION['challenge'] = $WebAuthn->getChallenge();
  } else if ($fn === 'getGetArgs') {
    // ------------------------------------
    // request for get arguments
    // ------------------------------------

    $ids = [];

    // load registrations from session stored there by processCreate.
    // normaly you have to load the credential Id's for a username
    // from the database.
    if (isset($_PERSISTENT['registrations']) && is_array($_PERSISTENT['registrations'])) {
      foreach ($_PERSISTENT['registrations'] as $reg) {
        if ($reg->userId === $userId) {
          $ids[] = $reg->credentialId;
        }
      }
    }

    if (count($ids) === 0) {
      throw new Exception('no registrations in session for userId ' . $userId);
    }

    $getArgs = $WebAuthn->getGetArgs($ids, 20, true, true, true, true, true, 'discouraged');

    header('Content-Type: application/json');
    print(json_encode($getArgs));

    // save challange to session. you have to deliver it to processGet later.
    $_SESSION['challenge'] = $WebAuthn->getChallenge();
  } else if ($fn === 'processCreate') {
    // ------------------------------------
    // process create
    // ------------------------------------

    $clientDataJSON = base64_decode($post->clientDataJSON);
    $attestationObject = base64_decode($post->attestationObject);
    $challenge = $_SESSION['challenge'];

    // processCreate returns data to be stored for future logins.
    // in this example we store it in the php session.
    // Normaly you have to store the data in a database connected
    // with the user name.
    $data = $WebAuthn->processCreate($clientDataJSON, $attestationObject, $challenge, false, true, false);

    // add user infos
    $data->userId = $userId;
    $data->userName = $userName;
    $data->userDisplayName = $userDisplayName;

    if (!isset($_PERSISTENT['registrations']) || !array_key_exists('registrations', $_PERSISTENT) || !is_array($_PERSISTENT['registrations'])) {
      $_PERSISTENT['registrations'] = [];
    }
    $_PERSISTENT['registrations'][] = $data;

    $msg = 'registration success.';
    if ($data->rootValid === false) {
      $msg = 'registration ok, but certificate does not match any of the selected root ca.';
    }

    $return = new stdClass();
    $return->success = true;
    $return->msg = $msg;

    header('Content-Type: application/json');
    print(json_encode($return));
  } else if ($fn === 'processGet') {
    // ------------------------------------
    // proccess get
    // ------------------------------------

    $clientDataJSON = base64_decode($post->clientDataJSON);
    $authenticatorData = base64_decode($post->authenticatorData);
    $signature = base64_decode($post->signature);
    $id = base64_decode($post->id);
    $challenge = $_SESSION['challenge'] ?? '';
    $credentialPublicKey = null;

    // looking up correspondending public key of the credential id
    // you should also validate that only ids of the given user name
    // are taken for the login.
    if (isset($_PERSISTENT['registrations']) && is_array($_PERSISTENT['registrations'])) {
      foreach ($_PERSISTENT['registrations'] as $reg) {
        if ($reg->credentialId === $id) {
          $credentialPublicKey = $reg->credentialPublicKey;
          break;
        }
      }
    }

    if ($credentialPublicKey === null) {
      throw new Exception('Public Key for credential ID not found!');
    }

    // process the get request. throws WebAuthnException if it fails
    $WebAuthn->processGet($clientDataJSON, $authenticatorData, $signature, $credentialPublicKey, $challenge, false, false);

    $return = new stdClass();
    $return->success = true;

    header('Content-Type: application/json');
    if (getenv("WEBAUTHN_SUCCESS_INCLUDE") && file_exists(getenv("WEBAUTHN_SUCCESS_INCLUDE"))) {
      include(getenv("WEBAUTHN_SUCCESS_INCLUDE"));
    }

    print(json_encode($return));
  } else if ($fn === 'clearRegistrations') {
    // ------------------------------------
    // proccess clear registrations
    // ------------------------------------

    $_PERSISTENT['registrations'] = null;
    $_SESSION['challenge'] = null;

    $return = new stdClass();
    $return->success = true;
    $return->msg = 'all registrations deleted';

    header('Content-Type: application/json');
    print(json_encode($return));
  }
} catch (Throwable $ex) {
  $return = new stdClass();
  $return->success = false;
  $return->msg = $ex->getMessage();

  header('Content-Type: application/json');
  print(json_encode($return));
}

@persistent_end();
