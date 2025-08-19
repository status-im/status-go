from clients.statusgo_container import PushNotificationServerContainer


class PushNotificationServer:
    container = None

    def __init__(self, gorush_port=8080):
        self.gorush_port = gorush_port
        self.container = PushNotificationServerContainer(
            identity="3e64442a0ba8a59b4d2dc7385cd4533a10e86dd644e7ec549cb92503787f5282",
            gorush_port=self.gorush_port,
        )

        self.data_dir = self.container.data_dir()
        self.container.start_health_monitoring()

        assert self.data_dir != ""
