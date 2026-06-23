FROM python:3.12-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PYTHONPATH=/opt/example

WORKDIR /opt/agent

COPY adapters/agents-sdk/pyproject.toml /tmp/adapter/pyproject.toml
COPY adapters/agents-sdk/src /tmp/adapter/src
RUN pip install --no-cache-dir /tmp/adapter

COPY examples/agents-sdk/agent.py /opt/example/example_agent.py

RUN useradd --create-home --uid 10001 agentos
USER agentos

ENV AGENTOS_AGENT_FACTORY=example_agent:create_agent

ENTRYPOINT ["agentos-agents-sdk"]

