FROM scratch
ADD rfid-alert-service /
EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s CMD ["/rfid-alert-service","-isHealthy"]
ENTRYPOINT ["/rfid-alert-service"]

