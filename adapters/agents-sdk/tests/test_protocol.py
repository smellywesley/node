import json
import unittest

from agentos_agents_sdk.protocol import (
    PROTOCOL_VERSION,
    ProtocolError,
    encode_message,
    json_safe,
    message,
    parse_message,
)


class Dumpable:
    def model_dump(self, mode):
        return {"mode": mode, "value": 7}


class ProtocolTests(unittest.TestCase):
    def test_parse_requires_object_and_type(self):
        with self.assertRaises(ProtocolError):
            parse_message("[]")
        with self.assertRaises(ProtocolError):
            parse_message('{"id":"missing-type"}')

    def test_message_and_encoding_are_json_lines_safe(self):
        payload = message("result", output=Dumpable())
        encoded = encode_message(payload)
        decoded = json.loads(encoded)

        self.assertNotIn("\n", encoded)
        self.assertEqual(decoded["protocol"], PROTOCOL_VERSION)
        self.assertEqual(decoded["output"], {"mode": "json", "value": 7})

    def test_json_safe_handles_nested_values(self):
        value = json_safe({"items": (1, object())})
        self.assertEqual(value["items"][0], 1)
        self.assertIsInstance(value["items"][1], str)


if __name__ == "__main__":
    unittest.main()

