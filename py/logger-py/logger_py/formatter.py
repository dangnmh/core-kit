from __future__ import annotations

import datetime
import json
import logging
import re
import sys
from typing import Any

# Pattern to strip any emoji or unicode decorative symbols
_EMOJI_PATTERN = re.compile(
    "[\U0001f000-\U0001faff\u2600-\u27bf\u2300-\u23ff\u2b50\ufe0f\u200d\u203c\u2049\u2139]+",
    flags=re.UNICODE,
)

_RESERVED_RECORD_ATTRS = {
    "args",
    "asctime",
    "created",
    "exc_info",
    "exc_text",
    "filename",
    "funcName",
    "levelname",
    "levelno",
    "lineno",
    "module",
    "msecs",
    "message",
    "msg",
    "name",
    "pathname",
    "process",
    "processName",
    "relativeCreated",
    "stack_info",
    "thread",
    "threadName",
    "taskName",
    "service",
    "extra_fields",
    "context",
}


def _strip_emojis(text: str) -> str:
    """Removes emoji and decorative symbols from text, preserving intentional whitespace."""
    if not text:
        return text
    cleaned = _EMOJI_PATTERN.sub("", text)
    if not text.startswith(" ") and cleaned.startswith(" "):
        cleaned = cleaned.lstrip()
    if not text.endswith(" ") and cleaned.endswith(" "):
        cleaned = cleaned.rstrip()
    return cleaned


class JsonFormatter(logging.Formatter):
    """
    Formats LogRecord objects into single-line structured JSON with schema:
    - timestamp: ISO-8601 UTC string (e.g. 2026-09-21T18:10:00.123Z)
    - severity: record.levelname
    - service: self.service_name (or record.service if provided)
    - logger: record.name
    - message: record.getMessage() (without icons or emojis)
    - caller: f"{record.filename}:{record.lineno}"
    - trace_id: W3C trace id (if present)
    - span_id: W3C span id (if present)
    - context: record.extra_fields or extra context (if present)
    - exception: formatted traceback (if exc_info is present)
    """

    def __init__(
        self,
        service_name: str = "app",
        datefmt: str | None = None,
    ) -> None:
        super().__init__(datefmt=datefmt)
        self.service_name = service_name

    def format(self, record: logging.LogRecord) -> str:
        dt = datetime.datetime.fromtimestamp(record.created, tz=datetime.UTC)
        timestamp = dt.strftime("%Y-%m-%dT%H:%M:%S.%f")[:-3] + "Z"

        service = getattr(record, "service", None) or self.service_name

        try:
            raw_message = record.getMessage()
        except (TypeError, ValueError, KeyError, AttributeError):
            raw_message = str(record.msg)
        message = _strip_emojis(raw_message)

        caller = f"{record.filename}:{record.lineno}"

        payload: dict[str, Any] = {
            "timestamp": timestamp,
            "severity": record.levelname,
            "service": service,
            "logger": record.name,
            "message": message,
            "caller": caller,
        }

        # Hoist trace_id and span_id if present in record
        trace_id = getattr(record, "trace_id", None)
        span_id = getattr(record, "span_id", None)

        context = None
        if hasattr(record, "extra_fields") and record.extra_fields is not None:
            context = dict(record.extra_fields)
        elif hasattr(record, "context") and record.context is not None:
            context = dict(record.context)
        else:
            extras = {
                k: v
                for k, v in record.__dict__.items()
                if k not in _RESERVED_RECORD_ATTRS and not k.startswith("_")
            }
            if extras:
                context = extras

        if context:
            if "trace_id" in context:
                popped_trace = context.pop("trace_id")
                if not trace_id:
                    trace_id = popped_trace
            if "span_id" in context:
                popped_span = context.pop("span_id")
                if not span_id:
                    span_id = popped_span

        if trace_id:
            payload["trace_id"] = str(trace_id)
        if span_id:
            payload["span_id"] = str(span_id)

        if context:
            payload["context"] = context

        if record.exc_info:
            payload["exception"] = self.formatException(record.exc_info)
        elif getattr(record, "exc_text", None):
            payload["exception"] = record.exc_text

        return json.dumps(payload, ensure_ascii=False, default=str)


class ServiceLoggerAdapter(logging.LoggerAdapter):
    """LoggerAdapter that attaches a default service name and merges extra kwargs."""

    def __init__(self, logger: logging.Logger, service: str) -> None:
        super().__init__(logger, {"service": service})
        self.service = service

    def process(self, msg: Any, kwargs: dict[str, Any]) -> tuple[Any, dict[str, Any]]:
        extra = dict(self.extra) if self.extra else {}
        if kwargs.get("extra"):
            extra.update(kwargs["extra"])
        kwargs["extra"] = extra
        return msg, kwargs


def setup_logging(
    service_name: str,
    level: str | int = "INFO",
    stream: Any | None = None,
) -> logging.Logger:
    """
    Configures the root logger with a StreamHandler using JsonFormatter.
    Clears preexisting handlers to avoid duplicate output.
    """
    if isinstance(level, str):
        level_str = level.strip().upper()
        if not level_str:
            numeric_level = logging.INFO
        else:
            numeric_level = getattr(logging, level_str, logging.INFO)
    else:
        numeric_level = level

    root_logger = logging.getLogger()
    root_logger.setLevel(numeric_level)

    for handler in list(root_logger.handlers):
        root_logger.removeHandler(handler)

    stream_handler = logging.StreamHandler(stream or sys.stdout)
    stream_handler.setLevel(numeric_level)
    stream_handler.setFormatter(JsonFormatter(service_name=service_name))
    root_logger.addHandler(stream_handler)

    return root_logger


def get_logger(
    name: str,
    service: str | None = None,
) -> logging.Logger | ServiceLoggerAdapter:
    """
    Returns a logger by name. If service is provided, wraps the logger
    in a ServiceLoggerAdapter to associate emitted records with the given service.
    """
    logger = logging.getLogger(name)
    if service:
        return ServiceLoggerAdapter(logger, service)
    return logger
