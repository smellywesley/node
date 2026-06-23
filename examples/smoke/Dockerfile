FROM python:3.12-alpine

WORKDIR /app
COPY examples/smoke/agent.py /app/agent.py
USER 65532:65532
CMD ["python", "/app/agent.py"]
