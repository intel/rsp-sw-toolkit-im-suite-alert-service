FROM scratch
ADD rfid-alert-service /
EXPOSE 8080
ENTRYPOINT ["/rfid-rules-service"]
