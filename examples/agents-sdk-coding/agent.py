from collections.abc import AsyncIterator
from typing import Any

from agents import Agent
from agents.models.interface import Model, ModelResponse
from agents.usage import Usage
from openai.types.responses import ResponseOutputMessage, ResponseOutputText


ARTIFACT = """package reviewed

// Add returns the sum of two integers.
func Add(left, right int) int {
\treturn left + right
}
"""


class OfflineCodingModel(Model):
    async def get_response(
        self,
        system_instructions: str | None,
        input: Any,
        model_settings: Any,
        tools: list[Any],
        output_schema: Any,
        handoffs: list[Any],
        tracing: Any,
        *,
        previous_response_id: str | None,
        conversation_id: str | None,
        prompt: Any,
    ) -> ModelResponse:
        message = ResponseOutputMessage(
            id="offline-reviewed-artifact",
            content=[
                ResponseOutputText(
                    annotations=[],
                    text=ARTIFACT,
                    type="output_text",
                    logprobs=[],
                )
            ],
            role="assistant",
            status="completed",
            type="message",
        )
        return ModelResponse(
            output=[message],
            usage=Usage(requests=1, input_tokens=12, output_tokens=30, total_tokens=42),
            response_id="offline-coding-model",
        )

    async def stream_response(self, *args: Any, **kwargs: Any) -> AsyncIterator[Any]:
        if False:
            yield None
        raise RuntimeError("offline coding model uses non-streaming execution")


def create_agent() -> Agent:
    return Agent(
        name="Offline reviewed coding agent",
        instructions="Produce the requested reviewed Go source artifact.",
        model=OfflineCodingModel(),
    )
