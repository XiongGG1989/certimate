import { getI18n, useTranslation } from "react-i18next";
import { Form, Input } from "antd";
import { createSchemaFieldRule } from "antd-zod";
import { z } from "zod";

import MultipleSplitValueInput from "@/components/MultipleSplitValueInput";

import { useFormNestedFieldsContext } from "./_context";

const MULTIPLE_INPUT_SEPARATOR_PATTERN = /[;,，；\r\n]/u;

const BizDeployNodeConfigFieldsProviderUCloudUCDN = () => {
  const { i18n, t } = useTranslation();

  const { parentNamePath } = useFormNestedFieldsContext();
  const formSchema = z.object({
    [parentNamePath]: getSchema({ i18n }),
  });
  const formRule = createSchemaFieldRule(formSchema);
  const initialValues = getInitialValues();

  return (
    <>
      <Form.Item
        name={[parentNamePath, "endpoint"]}
        initialValue={initialValues.endpoint}
        label={t("workflow_node.deploy.form.ucloud_ucdn_endpoint.label")}
        rules={[formRule]}
        tooltip={<span dangerouslySetInnerHTML={{ __html: t("workflow_node.deploy.form.ucloud_ucdn_endpoint.tooltip") }}></span>}
      >
        <Input allowClear placeholder={t("workflow_node.deploy.form.ucloud_ucdn_endpoint.placeholder")} />
      </Form.Item>

      <Form.Item
        name={[parentNamePath, "domains"]}
        initialValue={initialValues.domains}
        label={t("workflow_node.deploy.form.ucloud_ucdn_domains.label")}
        extra={t("workflow_node.deploy.form.ucloud_ucdn_domains.help")}
        rules={[formRule]}
        tooltip={<span dangerouslySetInnerHTML={{ __html: t("workflow_node.deploy.form.ucloud_ucdn_domains.tooltip") }}></span>}
      >
        <MultipleSplitValueInput
          modalTitle={t("workflow_node.deploy.form.ucloud_ucdn_domains.multiple_input_modal.title")}
          placeholder={t("workflow_node.deploy.form.ucloud_ucdn_domains.placeholder")}
          placeholderInModal={t("workflow_node.deploy.form.ucloud_ucdn_domains.multiple_input_modal.placeholder")}
          splitOptions={{ removeEmpty: true, trimSpace: true }}
        />
      </Form.Item>
    </>
  );
};

const getInitialValues = (): Nullish<z.infer<ReturnType<typeof getSchema>>> => {
  return {
    domains: "",
  };
};

const getSchema = ({ i18n = getI18n() }: { i18n?: ReturnType<typeof getI18n> }) => {
  const { t: _ } = i18n;

  return z.object({
    endpoint: z.string().nullish(),
    domains: z.string().refine((v) => v.split(MULTIPLE_INPUT_SEPARATOR_PATTERN).some((e) => e.trim().length > 0)),
  });
};

const _default = Object.assign(BizDeployNodeConfigFieldsProviderUCloudUCDN, {
  getInitialValues,
  getSchema,
});

export default _default;
