((mod) => {

  const params = {
    userId: '737072696e676861636b',
    userName: 'springhack',
    userDisplayName: 'SpringHack'
  };

  function init(id, name, displayName) {
    params.userId = id;
    params.userName = name;
    params.userDisplayName = displayName;
  }

  function getGetParams() {
    let url = '';
    url += '&userId=' + params.userId;
    url += '&userName=' + params.userName;
    url += '&userDisplayName=' + params.userDisplayName;
    return url;
  }

  async function register() {
    // get create args
    let rep = await fetch('webauthn.php?fn=getCreateArgs' + getGetParams(), {method:'GET', cache:'no-cache'});
    const createArgs = await rep.json();

    // error handling
    if (createArgs.success === false) {
      throw new Error(createArgs.msg || 'unknown error occured');
    }

    recursiveBase64StrToArrayBuffer(createArgs);

    // create credentials
    const cred = await navigator.credentials.create(createArgs);

    // create object
    const authenticatorAttestationResponse = {
      transports: cred.response.getTransports  ? cred.response.getTransports() : null,
      clientDataJSON: cred.response.clientDataJSON  ? arrayBufferToBase64(cred.response.clientDataJSON) : null,
      attestationObject: cred.response.attestationObject ? arrayBufferToBase64(cred.response.attestationObject) : null
    };

    // check auth on server side
    rep = await fetch('webauthn.php?fn=processCreate' + getGetParams(), {
      method  : 'POST',
      body    : JSON.stringify(authenticatorAttestationResponse),
      cache   : 'no-cache'
    });
    const authenticatorAttestationServerResponse = await rep.json();

    // prompt server response
    if (authenticatorAttestationServerResponse.success) {
      return 'success';
    } else {
      throw new Error(authenticatorAttestationServerResponse.msg);
    }
  }

  async function authenticate() {
    // get check args
    let rep = await fetch('webauthn.php?fn=getGetArgs' + getGetParams(), {method:'GET',cache:'no-cache'});
    const getArgs = await rep.json();

    // error handling
    if (getArgs.success === false) {
      throw new Error(getArgs.msg);
    }

    recursiveBase64StrToArrayBuffer(getArgs);

    // check credentials with hardware
    const cred = await navigator.credentials.get(getArgs);

    // create object for transmission to server
    const authenticatorAttestationResponse = {
      id: cred.rawId ? arrayBufferToBase64(cred.rawId) : null,
      clientDataJSON: cred.response.clientDataJSON  ? arrayBufferToBase64(cred.response.clientDataJSON) : null,
      authenticatorData: cred.response.authenticatorData ? arrayBufferToBase64(cred.response.authenticatorData) : null,
      signature: cred.response.signature ? arrayBufferToBase64(cred.response.signature) : null,
      userHandle: cred.response.userHandle ? arrayBufferToBase64(cred.response.userHandle) : null
    };

    // send to server
    rep = await fetch('webauthn.php?fn=processGet' + getGetParams(), {
      method:'POST',
      body: JSON.stringify(authenticatorAttestationResponse),
      cache:'no-cache'
    });
    const authenticatorAttestationServerResponse = await rep.json();

    // check server response
    if (authenticatorAttestationServerResponse.success) {
      return 'success';
    } else {
      throw new Error(authenticatorAttestationServerResponse.msg);
    }
  }

  async function clearRegistration() {
    const resp = await fetch('webauthn.php?fn=clearRegistrations' + getGetParams(), {method:'GET',cache:'no-cache'});
    const json = await resp.json();
    if (json.success) {
      return 'success';
    } else {
      throw new Error(json.msg);
    }
  }

  function recursiveBase64StrToArrayBuffer(obj) {
    let prefix = '=?BINARY?B?';
    let suffix = '?=';
    if (typeof obj === 'object') {
      for (let key in obj) {
        if (typeof obj[key] === 'string') {
          let str = obj[key];
          if (str.substring(0, prefix.length) === prefix && str.substring(str.length - suffix.length) === suffix) {
            str = str.substring(prefix.length, str.length - suffix.length);
            let binary_string = window.atob(str);
            let len = binary_string.length;
            let bytes = new Uint8Array(len);
            for (let i = 0; i < len; i++)        {
              bytes[i] = binary_string.charCodeAt(i);
            }
            obj[key] = bytes.buffer;
          }
        } else {
          recursiveBase64StrToArrayBuffer(obj[key]);
        }
      }
    }
  }

  function arrayBufferToBase64(buffer) {
    let binary = '';
    let bytes = new Uint8Array(buffer);
    let len = bytes.byteLength;
    for (let i = 0; i < len; i++) {
      binary += String.fromCharCode( bytes[ i ] );
    }
    return window.btoa(binary);
  }

  mod.webAuthn = {
    init,
    register,
    authenticate,
    clearRegistration
  };

})(globalThis);
