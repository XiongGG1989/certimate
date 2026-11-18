import { useMemo } from "react";
import { getI18n, useTranslation } from "react-i18next";
import { useRequest } from "ahooks";
import { App, Form, Input, Radio, Select } from "antd";
import { createSchemaFieldRule } from "antd-zod";
import { z } from "zod";

import { listUCloudUCDNDomains } from "@/api/deployments";
import Show from "@/components/Show";
import { type WorkflowNodeConfigForBizApply, type WorkflowNodeConfigForBizUpload } from "@/domain/workflow";
import { useWorkflowStore } from "@/stores/workflow";
import { unwrapErrMsg } from "@/utils/error";
import { getCertificateSubjectAltNames } from "@/utils/x509";

import { useFormNestedFieldsContext, useNodeFormContext } from "./_context";
import { getAllPreviousNodes } from "../_util";
import { NodeType } from "../nodes/typings";

const DOMAIN_MATCH_PATTERN_EXACT = "exact" as const;
const DOMAIN_MATCH_PATTERN_CERTSAN = "certsan" as const;

const MULTIPLE_INPUT_SEPARATOR_PATTERN = /[;,，；\r\n]/u;
const INLINE_INPUT_SEPARATOR_PATTERN = /[;,，；]/gu;

const normalizeDomainsInput = (value?: string | null) => (value ?? "").replace(INLINE_INPUT_SEPARATOR_PATTERN, "\n").replace(/\r\n?/gu, "\n");
const splitDomainsInput = (value?: string | null) =>
  normalizeDomainsInput(value)
    .split("\n")
    .map((domain) => domain.trim())
    .filter(Boolean);

type CertificateDomainFilter = {
  filterByCertificate: boolean;
  certificateId?: string;
  certificateWorkflowId?: string;
  certificateOutputNodeId?: string;
  subjectAltNames?: string[];
};

const BizDeployNodeConfigFieldsProviderUCloudUCDN = () => {
  const { i18n, t } = useTranslation();
  const { notification } = App.useApp();

  const { parentNamePath } = useFormNestedFieldsContext();
  const { node } = useNodeFormContext();
  const workflowId = useWorkflowStore((state) => state.workflow.id);
  const formSchema = z.object({
    [parentNamePath]: getSchema({ i18n }),
  });
  const formRule = createSchemaFieldRule(formSchema);
  const formInst = Form.useFormInstance();
  const initialValues = getInitialValues();

  const fieldCertificateId = Form.useWatch("certificateId", { form: formInst, preserve: true });
  const fieldCertificateOutputNodeId = Form.useWatch("certificateOutputNodeId", { form: formInst, preserve: true });
  const fieldProviderAccessId = Form.useWatch("providerAccessId", { form: formInst, preserve: true });
  const fieldEndpoint = Form.useWatch([parentNamePath, "endpoint"], { form: formInst, preserve: true });
  const fieldDomainMatchPattern = Form.useWatch([parentNamePath, "domainMatchPattern"], { form: formInst, preserve: true });

  const certificateDomainFilter = useMemo<CertificateDomainFilter>(() => {
    if (fieldCertificateId) {
      return { filterByCertificate: true, certificateId: fieldCertificateId };
    }
    if (!fieldCertificateOutputNodeId) {
      return { filterByCertificate: false };
    }

    const certificateNode = getAllPreviousNodes(node).find((candidate) => candidate.id === fieldCertificateOutputNodeId);
    const fallback: CertificateDomainFilter = {
      filterByCertificate: true,
      certificateWorkflowId: workflowId,
      certificateOutputNodeId: fieldCertificateOutputNodeId,
    };
    if (!certificateNode) {
      return fallback;
    }

    if (certificateNode.flowNodeType === NodeType.BizUpload) {
      const config = certificateNode.form?.getValueIn("config") as WorkflowNodeConfigForBizUpload | undefined;
      if (config?.source === "form") {
        return {
          ...fallback,
          subjectAltNames: getCertificateSubjectAltNames(config.certificate ?? ""),
        };
      }
      if (config?.source === "library") {
        return {
          ...fallback,
          certificateId: config.certificateId,
        };
      }
      return fallback;
    }

    if (certificateNode.flowNodeType === NodeType.BizApply) {
      const config = certificateNode.form?.getValueIn("config") as WorkflowNodeConfigForBizApply | undefined;
      const subjectAltNames = config?.identifier === "domain" ? splitDomainsInput(config.domains) : [];
      if (subjectAltNames.length > 0) {
        return { ...fallback, subjectAltNames };
      }
    }

    return fallback;
  }, [fieldCertificateId, fieldCertificateOutputNodeId, node, workflowId]);

  const {
    data: selectableDomains = [],
    loading: selectableDomainsLoading,
    refresh: refreshSelectableDomains,
  } = useRequest(
    () =>
      listUCloudUCDNDomains({
        accessId: fieldProviderAccessId!,
        endpoint: fieldEndpoint,
        ...certificateDomainFilter,
      }),
    {
      ready: fieldDomainMatchPattern === DOMAIN_MATCH_PATTERN_CERTSAN && !!fieldProviderAccessId,
      refreshDeps: [fieldProviderAccessId, fieldEndpoint, fieldDomainMatchPattern, certificateDomainFilter],
      debounceWait: 300,
      onSuccess: (domains) => {
        if (!certificateDomainFilter.filterByCertificate) return;

        const fieldNamePath = [parentNamePath, "certSanDomains"];
        const selectedDomains = splitDomainsInput(formInst.getFieldValue(fieldNamePath));
        const selectableDomainSet = new Set(domains);
        const matchedSelectedDomains = selectedDomains.filter((domain) => selectableDomainSet.has(domain));
        if (matchedSelectedDomains.length !== selectedDomains.length) {
          formInst.setFieldValue(fieldNamePath, matchedSelectedDomains.join("\n"));
        }
      },
      onError: (err) => {
        notification.error({ title: t("common.text.request_error"), description: unwrapErrMsg(err) });
      },
    }
  );
  const selectableDomainOptions = useMemo(() => selectableDomains.map((domain) => ({ label: domain, value: domain })), [selectableDomains]);

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
        name={[parentNamePath, "domainMatchPattern"]}
        initialValue={initialValues.domainMatchPattern}
        label={t("workflow_node.deploy.form.shared_domain_match_pattern.label")}
        rules={[formRule]}
      >
        <Radio.Group
          options={[DOMAIN_MATCH_PATTERN_EXACT, DOMAIN_MATCH_PATTERN_CERTSAN].map((s) => ({
            label: t(`workflow_node.deploy.form.shared_domain_match_pattern.option.${s}.label`),
            value: s,
          }))}
        />
      </Form.Item>

      <Show when={fieldDomainMatchPattern !== DOMAIN_MATCH_PATTERN_CERTSAN}>
        <Form.Item
          name={[parentNamePath, "exactDomains"]}
          initialValue={initialValues.exactDomains}
          label={t("workflow_node.deploy.form.ucloud_ucdn_domains.label")}
          extra={t("workflow_node.deploy.form.ucloud_ucdn_domains.help")}
          getValueProps={(value) => ({ value: normalizeDomainsInput(value) })}
          normalize={normalizeDomainsInput}
          rules={[formRule]}
          tooltip={<span dangerouslySetInnerHTML={{ __html: t("workflow_node.deploy.form.ucloud_ucdn_domains.tooltip") }}></span>}
        >
          <Input.TextArea autoSize={{ minRows: 3, maxRows: 8 }} />
        </Form.Item>
      </Show>

      <Show when={fieldDomainMatchPattern === DOMAIN_MATCH_PATTERN_CERTSAN}>
        <Form.Item
          name={[parentNamePath, "certSanDomains"]}
          initialValue={initialValues.certSanDomains}
          label={t("workflow_node.deploy.form.ucloud_ucdn_domains.label")}
          extra={t("workflow_node.deploy.form.ucloud_ucdn_domains.cert_san_help")}
          getValueProps={(value) => ({ value: splitDomainsInput(value) })}
          normalize={(value: string[]) => value.join("\n")}
          rules={[formRule]}
          tooltip={<span dangerouslySetInnerHTML={{ __html: t("workflow_node.deploy.form.ucloud_ucdn_domains.tooltip") }}></span>}
        >
          <Select
            allowClear
            disabled={!fieldProviderAccessId}
            loading={selectableDomainsLoading}
            mode="multiple"
            notFoundContent={
              certificateDomainFilter.filterByCertificate ? t("workflow_node.deploy.form.ucloud_ucdn_domains.no_certificate_matches") : undefined
            }
            optionFilterProp="label"
            options={selectableDomainOptions}
            placeholder={t("workflow_node.deploy.form.ucloud_ucdn_domains.select_placeholder")}
            showSearch
            onOpenChange={(open) => {
              if (open && fieldProviderAccessId) refreshSelectableDomains();
            }}
          />
        </Form.Item>
      </Show>
    </>
  );
};

const getInitialValues = (): Nullish<z.infer<ReturnType<typeof getSchema>>> => {
  return {
    domainMatchPattern: DOMAIN_MATCH_PATTERN_EXACT,
    exactDomains: "",
    certSanDomains: "",
  };
};

const getSchema = ({ i18n = getI18n() }: { i18n?: ReturnType<typeof getI18n> }) => {
  const { t: _ } = i18n;

  return z
    .object({
      endpoint: z.string().nullish(),
      domainMatchPattern: z.enum([DOMAIN_MATCH_PATTERN_EXACT, DOMAIN_MATCH_PATTERN_CERTSAN]).default(DOMAIN_MATCH_PATTERN_EXACT),
      exactDomains: z.string().nullish(),
      certSanDomains: z.string().nullish(),
    })
    .superRefine((values, ctx) => {
      const fieldName = values.domainMatchPattern === DOMAIN_MATCH_PATTERN_CERTSAN ? "certSanDomains" : "exactDomains";
      const scDomains = z.string().refine((v) => v.split(MULTIPLE_INPUT_SEPARATOR_PATTERN).some((e) => e.trim().length > 0));
      const spDomains = scDomains.safeParse(values[fieldName]);
      if (!spDomains.success) {
        ctx.addIssue({
          code: "custom",
          message: z.treeifyError(spDomains.error).errors.join(),
          path: [fieldName],
        });
      }
    });
};

const _default = Object.assign(BizDeployNodeConfigFieldsProviderUCloudUCDN, {
  getInitialValues,
  getSchema,
});

export default _default;
