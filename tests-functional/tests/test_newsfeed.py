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
        assert backend.newsfeed_service.enabled() == True
        assert backend.newsfeed_service.notifications_enabled() == False
        assert backend.newsfeed_service.rss_enabled() == True

        # Test enabled setting
        backend.newsfeed_service.set_enabled(False)
        assert backend.newsfeed_service.enabled() == False
        backend.newsfeed_service.set_enabled(True)
        assert backend.newsfeed_service.enabled() == True

        # Test notifications_enabled setting
        backend.newsfeed_service.set_notifications_enabled(True)
        assert backend.newsfeed_service.notifications_enabled() == True
        backend.newsfeed_service.set_notifications_enabled(False)
        assert backend.newsfeed_service.notifications_enabled() == False

        # Test rss_enabled setting
        backend.newsfeed_service.set_rss_enabled(False)
        assert backend.newsfeed_service.rss_enabled() == False
        backend.newsfeed_service.set_rss_enabled(True)
        assert backend.newsfeed_service.rss_enabled() == True
