FROM germanorizzo/ws4sqlite AS ws4sqlite

FROM alpine
ADD init /init
ADD www /excalidraw/www
ADD storage.yaml /excalidraw/storage/storage.yaml
COPY --from=ws4sqlite /ws4sqlite /ws4sqlite
ENTRYPOINT ["/bin/sh", "/init"]
CMD ["-db", "/excalidraw/storage/storage.db", "-serve-dir", "/excalidraw/www"]
