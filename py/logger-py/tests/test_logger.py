import io
import json
import logging
import re
import sys

from logger_py import JsonFormatter, get_logger, setup_logging


def test_json_formatter_schema():
    formatter = JsonFormatter(service_name="test-service")
    logger = logging.getLogger("unit.schema")
    record = logger.makeRecord(
        name="unit.schema",
        level=logging.INFO,
        fn="test_logger.py",
        lno=20,
        msg="Simple log message",
        args=(),
        exc_info=None,
    )

    output = formatter.format(record)
    data = json.loads(output)

    assert "timestamp" in data
    assert data["severity"] == "INFO"
    assert data["service"] == "test-service"
    assert data["logger"] == "unit.schema"
    assert data["message"] == "Simple log message"
    assert data["caller"] == "test_logger.py:20"
    assert "context" not in data
    assert "exception" not in data


def test_json_formatter_timestamp_format():
    formatter = JsonFormatter(service_name="test-service")
    logger = logging.getLogger("unit.timestamp")
    record = logger.makeRecord(
        name="unit.timestamp",
        level=logging.DEBUG,
        fn="test_logger.py",
        lno=40,
        msg="Timestamp check",
        args=(),
        exc_info=None,
    )

    output = formatter.format(record)
    data = json.loads(output)

    # Format: YYYY-MM-DDTHH:MM:SS.mmmZ
    pattern = r"^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$"
    assert re.match(pattern, data["timestamp"]) is not None


def test_json_formatter_service_override():
    formatter = JsonFormatter(service_name="default-service")
    logger = logging.getLogger("unit.override")
    record = logger.makeRecord(
        name="unit.override",
        level=logging.WARNING,
        fn="test_logger.py",
        lno=55,
        msg="Overridden service",
        args=(),
        exc_info=None,
        extra={"service": "custom-service"},
    )

    output = formatter.format(record)
    data = json.loads(output)

    assert data["service"] == "custom-service"


def test_json_formatter_strips_emojis():
    formatter = JsonFormatter(service_name="test-service")
    logger = logging.getLogger("unit.emoji")

    # Message contains decorative unicode emojis
    msg_with_emojis = "\u26a0\ufe0f [VoiceEngine] Speech failed \U0001f6a8"
    record = logger.makeRecord(
        name="unit.emoji",
        level=logging.ERROR,
        fn="test_logger.py",
        lno=75,
        msg=msg_with_emojis,
        args=(),
        exc_info=None,
    )

    output = formatter.format(record)
    data = json.loads(output)

    # Ensure emojis are removed from the message
    assert "\u26a0" not in data["message"]
    assert "\U0001f6a8" not in data["message"]
    assert data["message"] == "[VoiceEngine] Speech failed"


def test_json_formatter_with_context():
    formatter = JsonFormatter(service_name="test-service")
    logger = logging.getLogger("unit.context")

    # Test with extra_fields
    record1 = logger.makeRecord(
        name="unit.context",
        level=logging.INFO,
        fn="test_logger.py",
        lno=95,
        msg="Context test 1",
        args=(),
        exc_info=None,
        extra={"extra_fields": {"request_id": "req-123", "user": "test"}},
    )
    data1 = json.loads(formatter.format(record1))
    assert data1["context"] == {"request_id": "req-123", "user": "test"}

    # Test with context attribute
    record2 = logger.makeRecord(
        name="unit.context",
        level=logging.INFO,
        fn="test_logger.py",
        lno=107,
        msg="Context test 2",
        args=(),
        exc_info=None,
        extra={"context": {"user_id": "usr-456"}},
    )
    data2 = json.loads(formatter.format(record2))
    assert data2["context"] == {"user_id": "usr-456"}

    # Test with arbitrary extra kwargs
    record3 = logger.makeRecord(
        name="unit.context",
        level=logging.INFO,
        fn="test_logger.py",
        lno=119,
        msg="Context test 3",
        args=(),
        exc_info=None,
        extra={"custom_key": 999},
    )
    data3 = json.loads(formatter.format(record3))
    assert data3["context"] == {"custom_key": 999}


def test_json_formatter_with_exception():
    formatter = JsonFormatter(service_name="test-service")
    logger = logging.getLogger("unit.exception")

    try:
        raise ValueError("Critical test failure")
    except ValueError:
        exc_info = sys.exc_info()

    record = logger.makeRecord(
        name="unit.exception",
        level=logging.CRITICAL,
        fn="test_logger.py",
        lno=140,
        msg="Error occurred during execution",
        args=(),
        exc_info=exc_info,
    )

    output = formatter.format(record)
    data = json.loads(output)

    assert "exception" in data
    assert "ValueError: Critical test failure" in data["exception"]
    assert "Traceback (most recent call last)" in data["exception"]


def test_setup_logging_and_get_logger():
    stream = io.StringIO()
    setup_logging(service_name="suite-app", level="INFO", stream=stream)

    log = get_logger("suite.client")
    log.info("System initialized successfully", extra={"context": {"mode": "test"}})

    log_output = stream.getvalue().strip()
    assert log_output != ""

    lines = log_output.splitlines()
    assert len(lines) == 1

    entry = json.loads(lines[0])
    assert entry["severity"] == "INFO"
    assert entry["service"] == "suite-app"
    assert entry["logger"] == "suite.client"
    assert entry["message"] == "System initialized successfully"
    assert entry["context"] == {"mode": "test"}


def test_get_logger_service_adapter():
    stream = io.StringIO()
    setup_logging(service_name="base-service", level="INFO", stream=stream)

    custom_log = get_logger("suite.subsystem", service="subsystem-service")
    custom_log.info("Subsystem heartbeat", extra={"context": {"status": "healthy"}})

    lines = stream.getvalue().strip().splitlines()
    assert len(lines) == 1

    entry = json.loads(lines[0])
    assert entry["service"] == "subsystem-service"
    assert entry["logger"] == "suite.subsystem"
    assert entry["message"] == "Subsystem heartbeat"
    assert entry["context"] == {"status": "healthy"}


def test_json_formatter_hoists_trace_ids():
    formatter = JsonFormatter(service_name="test-worker")
    record = logging.LogRecord(
        name="inference",
        level=logging.INFO,
        pathname="worker.py",
        lineno=42,
        msg="Execution completed",
        args=(),
        exc_info=None,
    )
    record.trace_id = "4bf92f3577b34da6a3ce929d0e0e4736"
    record.span_id = "00f067aa0ba902b7"
    record.extra_fields = {"tokens": 120}

    line = formatter.format(record)
    data = json.loads(line)
    assert data["trace_id"] == "4bf92f3577b34da6a3ce929d0e0e4736"
    assert data["span_id"] == "00f067aa0ba902b7"
    assert data["context"] == {"tokens": 120}
    assert "trace_id" not in data.get("context", {})
    assert "span_id" not in data.get("context", {})


def test_json_formatter_hoists_trace_ids_from_extra_context():
    formatter = JsonFormatter(service_name="test-worker")
    logger = logging.getLogger("unit.trace_extra")
    record = logger.makeRecord(
        name="unit.trace_extra",
        level=logging.INFO,
        fn="worker.py",
        lno=50,
        msg="Trace extra test",
        args=(),
        exc_info=None,
        extra={
            "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
            "span_id": "00f067aa0ba902b7",
            "batch_size": 50,
        },
    )

    line = formatter.format(record)
    data = json.loads(line)
    assert data["trace_id"] == "4bf92f3577b34da6a3ce929d0e0e4736"
    assert data["span_id"] == "00f067aa0ba902b7"
    assert data["context"] == {"batch_size": 50}
    assert "trace_id" not in data.get("context", {})
    assert "span_id" not in data.get("context", {})
