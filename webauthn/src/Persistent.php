<?php

function persistent_start() {
  session_start();
  global $_SESSION;
  if (file_exists('/data/persistent.db')) {
    $data = file_get_contents('/data/persistent.db');
    try {
      $_SESSION = unserialize($data);
    } catch (Throwable) {
      $_SESSION = [];
    }
  }
}

function persistent_end() {
  file_put_contents('/data/persistent.db', serialize($_SESSION));
}
