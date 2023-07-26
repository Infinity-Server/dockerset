<?php

$_PERSISTENT = [];

function persistent_start() {
  global $_PERSISTENT;
  session_name('webauthn_session');
  session_start();
  if (file_exists('/data/persistent.db')) {
    $data = file_get_contents('/data/persistent.db');
    try {
      $_PERSISTENT = unserialize($data);
    } catch (Throwable) {
      $_PERSISTENT = [];
    }
  }
}

function persistent_end() {
  global $_PERSISTENT;
  file_put_contents('/data/persistent.db', serialize($_PERSISTENT));
}
