from clients.services.linkpreview import URLUnfurlPermission
from steps.messenger import MessengerSteps


class TestLinkPreview(MessengerSteps):

    def test_contact_link_preview(self, backend_new_profile):
        user_1 = backend_new_profile("user_1")
        user_2 = backend_new_profile("user_2")
        self.make_contacts(user_1, user_2)

        user_1_url = user_1.sharedurls_service.share_user_url_with_data(user_1.public_key)
        text = f"Here's my shared URL: {user_1_url}"
        plan = user_1.linkpreview_service.get_text_urls_to_unfurl(text)
        assert plan is not None
        assert len(plan["urls"]) == 1

        metadata = plan["urls"][0]
        assert metadata["url"] == user_1_url
        assert metadata["isStatusSharedURL"] is True
        assert metadata["permission"] is URLUnfurlPermission.URLUnfurlingAllowed.value
