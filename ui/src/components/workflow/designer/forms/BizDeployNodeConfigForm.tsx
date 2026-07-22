import { useEffect, useMemo } from "react";
import { getI18n, useTranslation } from "react-i18next";
import { type FlowNodeEntity } from "@flowgram.ai/fixed-layout-editor";
import { IconPlus } from "@tabler/icons-react";
import { useRequest } from "ahooks";
import { type AnchorProps, Button, Divider, Form, type FormInstance, Radio, Select, Switch, Typography, theme } from "antd";
import { createSchemaFieldRule } from "antd-zod";
import { z } from "zod";

import AccessEditDrawer from "@/components/access/AccessEditDrawer";
import AccessSelect from "@/components/access/AccessSelect";
import DeploymentProviderPicker from "@/components/provider/DeploymentProviderPicker";
import DeploymentProviderSelect from "@/components/provider/DeploymentProviderSelect";
import Show from "@/components/Show";
import { type AccessModel } from "@/domain/access";
import { type CertificateModel } from "@/domain/certificate";
import { deploymentProvidersMap } from "@/domain/provider";
import { type WorkflowNodeConfigForBizDeploy, defaultNodeConfigForBizDeploy } from "@/domain/workflow";
import { useAntdForm, useZustandShallowSelector } from "@/hooks";
import { useAccessesStore } from "@/stores/access";
import { listManualUploads } from "@/repository/certificate";

import { getAllPreviousNodes } from "../_util";
import { FormNestedFieldsContextProvider, NodeFormContextProvider } from "./_context";
import BizDeployNodeConfigFieldsProvider from "./BizDeployNodeConfigFieldsProvider";
import { NodeType } from "../nodes/typings";

export interface BizDeployNodeConfigFormProps {
  form: FormInstance;
  node: FlowNodeEntity;
}

const CERTIFICATE_SOURCE_NODE = "node" as const;
const CERTIFICATE_SOURCE_MANUAL = "manual" as const;

const BizDeployNodeConfigForm = ({ node, ...props }: BizDeployNodeConfigFormProps) => {
  if (node.flowNodeType !== NodeType.BizDeploy) {
    console.warn(`[certimate] current workflow node type is not: ${NodeType.BizDeploy}`);
  }

  const { i18n, t } = useTranslation();

  const { token: themeToken } = theme.useToken();

  const { accesses } = useAccessesStore(useZustandShallowSelector("accesses"));
  const accessOptionFilter = (_: string, option: AccessModel) => {
    if (option.reserve) return false;
    return deploymentProvidersMap.get(fieldProvider)?.provider === option.provider;
  };

  const initialValues = useMemo(() => {
    const values = node.form?.getValueIn("config") as WorkflowNodeConfigForBizDeploy | undefined;
    if (!values) return values;
    return {
      ...values,
      certificateSource: values.certificateSource ?? (values.certificateId ? CERTIFICATE_SOURCE_MANUAL : CERTIFICATE_SOURCE_NODE),
    };
  }, [node]);

  const formSchema = getSchema({ i18n }).superRefine((values, ctx) => {
    if (values.certificateSource === CERTIFICATE_SOURCE_NODE && values.certificateOutputNodeId) {
      if (!certificateOutputNodeIdOptions.some((option) => option.value === values.certificateOutputNodeId)) {
        ctx.addIssue({
          code: "custom",
          message: t("workflow_node.deploy.form.certificate_output_node_id.placeholder"),
          path: ["certificateOutputNodeId"],
        });
      }
    }
  });
  const formRule = createSchemaFieldRule(formSchema);
  const { form: formInst, formProps } = useAntdForm<z.infer<typeof formSchema>>({
    form: props.form,
    name: "workflowNodeBizDeployConfigForm",
    initialValues: initialValues ?? getInitialValues(),
  });

  const fieldProvider = Form.useWatch("provider", { form: formInst, preserve: true });
  const fieldCertificateSource = Form.useWatch("certificateSource", { form: formInst, preserve: true });
  const fieldProviderAccessId = Form.useWatch("providerAccessId", { form: formInst, preserve: true });

  const certificateOutputNodeIdOptions = useMemo(() => {
    return getAllPreviousNodes(node)
      .filter((node) => node.flowNodeType === NodeType.BizApply || node.flowNodeType === NodeType.BizUpload)
      .map((node) => {
        return {
          label: node.form?.getValueIn("name"),
          value: node.id,
        };
      });
  }, [node]);

  const { data: manualCertificates = [], loading: manualCertificatesLoading } = useRequest(listManualUploads);
  const manualCertificateOptions = useMemo(
    () =>
      manualCertificates.map((certificate) => ({
        label: certificate.subjectAltNames.split(";").join("; "),
        value: certificate.id,
        certificate,
      })),
    [manualCertificates]
  );

  const handleCertificateSourceChange = (value: string) => {
    if (value === CERTIFICATE_SOURCE_NODE) {
      formInst.setFieldValue("certificateId", void 0);
    } else {
      formInst.setFieldValue("certificateOutputNodeId", void 0);
    }
  };

  const renderNestedFieldProviderComponent = BizDeployNodeConfigFieldsProvider.useComponent(fieldProvider, {});

  const showProviderAccess = useMemo(() => {
    // 内置的部署提供商（如本地部署）无需显示授权信息字段
    if (fieldProvider) {
      const provider = deploymentProvidersMap.get(fieldProvider);
      return !provider?.builtin;
    }

    return false;
  }, [fieldProvider]);

  useEffect(() => {
    // 如果未选择提供商，则清空授权信息
    if (!fieldProvider && fieldProviderAccessId) {
      formInst.setFieldValue("providerAccessId", void 0);
      return;
    }

    // 如果已选择提供商只有一个授权信息，则自动选择该授权信息
    if (fieldProvider && !fieldProviderAccessId) {
      const availableAccesses = accesses
        .filter((access) => accessOptionFilter(access.provider, access))
        .filter((access) => deploymentProvidersMap.get(fieldProvider)?.provider === access.provider);
      if (availableAccesses.length === 1) {
        formInst.setFieldValue("providerAccessId", availableAccesses[0].id);
      }
    }
  }, [fieldProvider, fieldProviderAccessId]);

  const handleProviderPick = (value: string) => {
    formInst.setFieldValue("provider", value);
    formInst.setFieldValue("providerAccessId", void 0);
    formInst.setFieldValue("providerConfig", void 0);
  };

  const handleProviderSelect = (value?: string | undefined) => {
    // 切换提供商时重置表单，避免其他提供商的配置字段残留
    if (initialValues?.provider === value) {
      formInst.setFieldValue("providerAccessId", void 0);
      formInst.resetFields(["providerConfig"]);
    } else {
      formInst.setFieldValue("providerAccessId", void 0);
      formInst.setFieldValue("providerConfig", void 0);
    }
  };

  return (
    <NodeFormContextProvider value={{ node }}>
      <Form {...formProps} clearOnDestroy={true} form={formInst} layout="vertical" preserve={false} scrollToFirstError>
        <Show when={!fieldProvider}>
          <DeploymentProviderPicker
            placeholder={t("workflow_node.deploy.form.provider.search.placeholder")}
            showAvailability
            showSearch
            onSelect={handleProviderPick}
          />
        </Show>

        <div style={{ display: fieldProvider ? "block" : "none" }}>
          <div id="parameters" data-anchor="parameters">
            <Form.Item name="certificateSource" label={t("workflow_node.deploy.form.certificate_source.label")} rules={[formRule]}>
              <Radio.Group block onChange={(event) => handleCertificateSourceChange(event.target.value)}>
                <Radio.Button value={CERTIFICATE_SOURCE_NODE}>{t("workflow_node.deploy.form.certificate_source.option.node")}</Radio.Button>
                <Radio.Button value={CERTIFICATE_SOURCE_MANUAL}>{t("workflow_node.deploy.form.certificate_source.option.manual")}</Radio.Button>
              </Radio.Group>
            </Form.Item>

            {fieldCertificateSource === CERTIFICATE_SOURCE_NODE && (
              <Form.Item
                name="certificateOutputNodeId"
                label={t("workflow_node.deploy.form.certificate_output_node_id.label")}
                extra={t("workflow_node.deploy.form.certificate_output_node_id.help")}
                rules={[formRule]}
              >
                <Select
                  optionRender={({ label, value }) => {
                    return (
                      <div className="flex items-center justify-between gap-4 overflow-hidden">
                        <div className="flex-1 truncate">{label}</div>
                        <div className="origin-right scale-90 font-mono text-xs" style={{ color: themeToken.colorTextSecondary }}>
                          (NodeID: {value})
                        </div>
                      </div>
                    );
                  }}
                  options={certificateOutputNodeIdOptions}
                  placeholder={t("workflow_node.deploy.form.certificate_output_node_id.placeholder")}
                />
              </Form.Item>
            )}

            {fieldCertificateSource === CERTIFICATE_SOURCE_MANUAL && (
              <Form.Item
                name="certificateId"
                label={t("workflow_node.deploy.form.certificate_id.label")}
                extra={t("workflow_node.deploy.form.certificate_id.help")}
                rules={[formRule]}
              >
                <Select
                  loading={manualCertificatesLoading}
                  optionFilterProp="label"
                  optionRender={({ data }) => {
                    const certificate = data.certificate as CertificateModel;
                    return (
                      <div className="flex items-center justify-between gap-4 overflow-hidden">
                        <div className="flex-1 truncate">{data.label}</div>
                        <div className="origin-right scale-90 font-mono text-xs" style={{ color: themeToken.colorTextSecondary }}>
                          #{certificate.id}
                        </div>
                      </div>
                    );
                  }}
                  options={manualCertificateOptions}
                  placeholder={t("workflow_node.deploy.form.certificate_id.placeholder")}
                  showSearch
                />
              </Form.Item>
            )}
          </div>

          <div id="deployment" data-anchor="deployment">
            <Divider size="small">
              <Typography.Text className="text-xs font-normal" type="secondary">
                {t("workflow_node.deploy.form_anchor.deployment.title")}
              </Typography.Text>
            </Divider>

            <Form.Item name="provider" label={t("workflow_node.deploy.form.provider.label")} rules={[formRule]}>
              <DeploymentProviderSelect
                allowClear
                disabled={!!initialValues?.provider}
                placeholder={t("workflow_node.deploy.form.provider.placeholder")}
                showAvailability
                showSearch
                onSelect={handleProviderSelect}
                onClear={handleProviderSelect}
              />
            </Form.Item>

            <Form.Item className="relative" hidden={!showProviderAccess} label={t("workflow_node.deploy.form.provider_access.label")}>
              <div className="absolute -top-1.5 right-0 -translate-y-full">
                <AccessEditDrawer
                  data={{ provider: deploymentProvidersMap.get(fieldProvider!)?.provider }}
                  mode="create"
                  trigger={
                    <Button size="small" type="link">
                      {t("workflow_node.deploy.form.provider_access.button")}
                      <IconPlus size="1.25em" />
                    </Button>
                  }
                  usage="hosting"
                  afterSubmit={(record) => {
                    if (!accessOptionFilter(record.provider, record)) return;
                    if (deploymentProvidersMap.get(fieldProvider!)?.provider !== record.provider) return;
                    formInst.setFieldValue("providerAccessId", record.id);
                  }}
                />
              </div>
              <Form.Item name="providerAccessId" dependencies={["provider"]} rules={[formRule]} noStyle>
                <AccessSelect
                  disabled={!fieldProvider}
                  placeholder={t("workflow_node.deploy.form.provider_access.placeholder")}
                  showSearch
                  onFilter={accessOptionFilter}
                />
              </Form.Item>
            </Form.Item>

            <FormNestedFieldsContextProvider value={{ parentNamePath: "providerConfig" }}>
              {renderNestedFieldProviderComponent && <>{renderNestedFieldProviderComponent}</>}
            </FormNestedFieldsContextProvider>
          </div>

          <div id="strategy" data-anchor="strategy">
            <Divider size="small">
              <Typography.Text className="text-xs font-normal" type="secondary">
                {t("workflow_node.deploy.form_anchor.strategy.title")}
              </Typography.Text>
            </Divider>

            <Form.Item label={t("workflow_node.deploy.form.skip_on_last_succeeded.label")}>
              <span className="me-2 inline-block">{t("workflow_node.deploy.form.skip_on_last_succeeded.prefix")}</span>
              <span className="inline-block">
                <Form.Item name="skipOnLastSucceeded" noStyle rules={[formRule]}>
                  <Switch
                    checkedChildren={t("workflow_node.deploy.form.skip_on_last_succeeded.switch.on")}
                    unCheckedChildren={t("workflow_node.deploy.form.skip_on_last_succeeded.switch.off")}
                  />
                </Form.Item>
              </span>
              <span className="ms-2 inline-block">{t("workflow_node.deploy.form.skip_on_last_succeeded.suffix")}</span>
            </Form.Item>
          </div>
        </div>
      </Form>
    </NodeFormContextProvider>
  );
};

const getAnchorItems = ({ i18n = getI18n() }: { i18n?: ReturnType<typeof getI18n> }): Required<AnchorProps>["items"] => {
  const { t } = i18n;

  return ["parameters", "deployment", "strategy"].map((key) => ({
    key: key,
    title: t(`workflow_node.deploy.form_anchor.${key}.tab`),
    href: "#" + key,
  }));
};

const getInitialValues = (): Nullish<z.infer<ReturnType<typeof getSchema>>> => {
  return {
    ...(defaultNodeConfigForBizDeploy() as Nullish<z.infer<ReturnType<typeof getSchema>>>),
    certificateSource: CERTIFICATE_SOURCE_NODE,
  };
};

const getSchema = ({ i18n = getI18n() }: { i18n?: ReturnType<typeof getI18n> }) => {
  const { t } = i18n;

  return z
    .object({
      certificateSource: z.enum([CERTIFICATE_SOURCE_NODE, CERTIFICATE_SOURCE_MANUAL]).default(CERTIFICATE_SOURCE_NODE),
      certificateOutputNodeId: z.string().nullish(),
      certificateId: z.string().nullish(),
      provider: z.string().nonempty(),
      providerAccessId: z.string().nullish(),
      providerConfig: z.any().nullish(),
      skipOnLastSucceeded: z.boolean().nullish(),
    })
    .superRefine((values, ctx) => {
      if (values.certificateSource === CERTIFICATE_SOURCE_NODE && !values.certificateOutputNodeId) {
        ctx.addIssue({
          code: "custom",
          message: t("workflow_node.deploy.form.certificate_output_node_id.placeholder"),
          path: ["certificateOutputNodeId"],
        });
      }
      if (values.certificateSource === CERTIFICATE_SOURCE_MANUAL && !values.certificateId) {
        ctx.addIssue({
          code: "custom",
          message: t("workflow_node.deploy.form.certificate_id.placeholder"),
          path: ["certificateId"],
        });
      }
      if (values.provider) {
        const provider = deploymentProvidersMap.get(values.provider);
        if (!provider?.builtin && !values.providerAccessId) {
          ctx.addIssue({
            code: "custom",
            message: t("workflow_node.deploy.form.provider_access.placeholder"),
            path: ["providerAccessId"],
          });
        }
      }
    });
};

const _default = Object.assign(BizDeployNodeConfigForm, {
  getAnchorItems,
  getSchema,
});

export default _default;
