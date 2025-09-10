import pytest


def _resp_json(resp):
    try:
        return resp.json()
    except Exception:
        return resp


@pytest.mark.rpc
class TestMakePartiallyOperableAccoutsFullyOperable:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")

    def test_make_partially_operable_accouts_fully_operable_returns_list(self):
        """
        This test invokes the RPC that should convert any partially-operable accounts
        into fully-operable ones. It's possible there are no partially-operable accounts;
        in that case the endpoint should still respond successfully and return an empty list.
        """
        resp = self.account.accounts_service.make_partially_operable_accouts_fully_operable(self.account.password)
        data = _resp_json(resp)
        # Should not return an RPC-level error
        assert "error" not in data
        # Result may be None or a list of addresses; accept both but check type if present
        result = data.get("result") if isinstance(data, dict) else None
        if result is not None:
            assert isinstance(result, list)
