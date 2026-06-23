FROM python:3.12-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_DEFAULT_TIMEOUT=60

WORKDIR /opt/agent

COPY adapters/agents-sdk/pyproject.toml /tmp/adapter/pyproject.toml
COPY adapters/agents-sdk/constraints.txt /tmp/adapter/constraints.txt
COPY adapters/agents-sdk/src /tmp/adapter/src
RUN pip install --no-cache-dir --constraint /tmp/adapter/constraints.txt /tmp/adapter

RUN useradd --create-home --uid 10001 agentos
USER agentos

ENTRYPOINT ["agentos-agents-sdk"]
