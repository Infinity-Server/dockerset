window.addEventListener('load', () => {
  client.callback_close = () => {
    location.reload();
  };
});
