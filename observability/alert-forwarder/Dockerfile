FROM python:3.12-alpine
WORKDIR /app
COPY forwarder.py .
RUN mkdir -p /var/log/alert-forwarder
CMD ["python3", "-u", "forwarder.py"]
