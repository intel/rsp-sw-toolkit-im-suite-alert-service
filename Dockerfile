FROM scratch
ADD rfid-alert-service /
EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s CMD ["/rfid-alert-service","-isHealthy"]

ARG GIT_COMMIT=unspecified
LABEL git_commit=$GIT_COMMIT

ENTRYPOINT ["/rfid-alert-service"]

