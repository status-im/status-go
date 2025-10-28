import pytest

from clients.signals import SignalType


@pytest.mark.rpc
class TestNewsFeed:

    await_signals = [
        SignalType.NODE_LOGIN.value,
        SignalType.NODE_STARTED.value,
        SignalType.NODE_READY.value,
    ]

    def test_newsfeed_settings(self, backend_new_profile):
        backend = backend_new_profile("backend")

        # Verify initial values
        assert backend.newsfeed_service.enabled()
        assert not backend.newsfeed_service.notifications_enabled()
        assert backend.newsfeed_service.rss_enabled()

        # Test enabled setting
        backend.newsfeed_service.set_enabled(False)
        assert not backend.newsfeed_service.enabled()
        backend.newsfeed_service.set_enabled(True)
        assert backend.newsfeed_service.enabled()

        # Test notifications_enabled setting
        backend.newsfeed_service.set_notifications_enabled(True)
        assert backend.newsfeed_service.notifications_enabled()
        backend.newsfeed_service.set_notifications_enabled(False)
        assert not backend.newsfeed_service.notifications_enabled()

        # Test rss_enabled setting
        backend.newsfeed_service.set_rss_enabled(False)
        assert not backend.newsfeed_service.rss_enabled()
        backend.newsfeed_service.set_rss_enabled(True)
        assert backend.newsfeed_service.rss_enabled()
