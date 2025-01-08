FROM germanorizzo/ws4sqlite
ADD . /excalidraw
WORKDIR /excalidraw
ENTRYPOINT ["/ws4sqlite"]
CMD ["-db", "/excalidraw/storage.db", "-serve-dir", "/excalidraw/www"]
