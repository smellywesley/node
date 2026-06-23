from __future__ import annotations

import argparse
import asyncio
import logging
import os

from .runtime import AdapterRuntime, OpenAIAgentsBackend


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Agent Process OS v1 adapter for the OpenAI Agents SDK"
    )
    parser.add_argument(
        "--heartbeat-seconds",
        type=float,
        default=float(os.getenv("AGENTOS_HEARTBEAT_SECONDS", "15")),
    )
    parser.add_argument(
        "--agent-factory",
        default=os.getenv("AGENTOS_AGENT_FACTORY"),
        help="Agent factory as module.path:callable",
    )
    return parser


async def async_main() -> None:
    args = build_parser().parse_args()
    runtime = AdapterRuntime(
        backend=OpenAIAgentsBackend(args.agent_factory),
        heartbeat_seconds=args.heartbeat_seconds,
    )
    await runtime.serve()


def main() -> None:
    logging.basicConfig(
        level=os.getenv("LOG_LEVEL", "INFO").upper(),
        format="%(asctime)s %(levelname)s %(name)s %(message)s",
    )
    asyncio.run(async_main())


if __name__ == "__main__":
    main()
