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
