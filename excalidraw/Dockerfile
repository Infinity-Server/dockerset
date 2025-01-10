FROM germanorizzo/ws4sqlite AS ws4sqlite

FROM alpine
ADD www /excalidraw/www
ADD init /excalidraw/init
ADD storage.yaml /excalidraw/storage.yaml
COPY --from=ws4sqlite /ws4sqlite /excalidraw/ws4sqlite

RUN mkdir /excalidraw/storage

WORKDIR /excalidraw
ENTRYPOINT ["/bin/sh", "/excalidraw/init"]
CMD ["-db", "/excalidraw/storage/storage.db", "-serve-dir", "/excalidraw/www"]
