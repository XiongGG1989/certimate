import { useTranslation } from "react-i18next";
import { IconBrandGithub, IconHeartHandshake } from "@tabler/icons-react";
import { Button, Divider, Typography } from "antd";

import { APP_REPO_URL, APP_VERSION } from "@/domain/app";

const SettingsAbout = () => {
  const { t } = useTranslation();

  return (
    <div className="max-w-3xl">
      <div className="flex items-center gap-5">
        <img src="/logo.svg" className="size-20" alt="AUTO_SSL" />
        <div>
          <h2 className="mb-1">AUTO_SSL</h2>
          <Typography.Text type="secondary">Version: {APP_VERSION}</Typography.Text>
        </div>
      </div>

      <Divider />

      <div className="flex items-start gap-4">
        <div className="mt-1 rounded-full bg-green-50 p-3 text-green-700">
          <IconHeartHandshake size="2em" />
        </div>
        <div>
          <h2 className="mb-2">{t("settings.about.acknowledgement.title")}</h2>
          <Typography.Paragraph type="secondary">{t("settings.about.acknowledgement.description")}</Typography.Paragraph>
          <Button icon={<IconBrandGithub size="1.25em" />} onClick={() => window.open(APP_REPO_URL, "_blank")}>
            {t("settings.about.acknowledgement.button")}
          </Button>
        </div>
      </div>
    </div>
  );
};

export default SettingsAbout;
