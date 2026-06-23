FROM python:3.12-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PYTHONPATH=/opt/example \
    PIP_DEFAULT_TIMEOUT=60

WORKDIR /opt/agent

COPY adapters/agents-sdk/pyproject.toml /tmp/adapter/pyproject.toml
COPY adapters/agents-sdk/constraints.txt /tmp/adapter/constraints.txt
COPY adapters/agents-sdk/src /tmp/adapter/src
RUN pip install --no-cache-dir --constraint /tmp/adapter/constraints.txt /tmp/adapter

COPY examples/agents-sdk-coding/agent.py /opt/example/coding_agent.py

RUN useradd --create-home --uid 10001 agentos
USER agentos

ENV AGENTOS_AGENT_FACTORY=coding_agent:create_agent
ENTRYPOINT ["agentos-agents-sdk"]
