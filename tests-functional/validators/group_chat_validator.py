class GroupChatValidator:
    def __init__(self, response_data):
        self.response_data = response_data

    def validate_fields(self, expected_fields):
        for field, expected_value in expected_fields.items():
            actual_value = self.response_data.get(field)
            if isinstance(expected_value, list):
                if not all(
                        any(member.get("id") == ev["id"] for member in actual_value)
                        for ev in expected_value
                ):
                    raise AssertionError(f"Validation failed: Mismatched members for field '{field}'")
            elif actual_value != expected_value:
                raise AssertionError(
                    f"Validation failed for field '{field}': expected '{expected_value}', got '{actual_value}'")

    def validate_message_fields(self, expected_fields):
        for field, expected_value in expected_fields.items():
            if field == "timestamp":
                continue

            actual_value = self.response_data.get(field)

            if isinstance(expected_value, dict) and isinstance(actual_value, dict):
                for sub_field, sub_value in expected_value.items():
                    if actual_value.get(sub_field) != sub_value:
                        raise AssertionError(
                            f"Validation failed for nested field '{field}.{sub_field}': expected '{sub_value}', got '{actual_value.get(sub_field)}'"
                        )
            elif isinstance(expected_value, list) and isinstance(actual_value, list):
                if not all(item in actual_value for item in expected_value):
                    raise AssertionError(f"Validation failed: Mismatched list items for field '{field}'")
            elif actual_value != expected_value:
                raise AssertionError(
                    f"Validation failed for field '{field}': expected '{expected_value}', got '{actual_value}'"
                )
