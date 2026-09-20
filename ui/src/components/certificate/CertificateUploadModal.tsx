import { useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { App, Form, Input } from "antd";
import { createSchemaFieldRule } from "antd-zod";
import { z } from "zod";

import { upload as uploadCertificate } from "@/api/certificates";
import FileTextInput from "@/components/FileTextInput";
import ModalForm from "@/components/ModalForm";
import { type CertificateModel } from "@/domain/certificate";
import { unwrapErrMsg } from "@/utils/error";
import { getCertificateSubjectAltNames, validatePEMCertificate, validatePEMPrivateKey } from "@/utils/x509";

export interface CertificateUploadModalProps {
  certificate?: CertificateModel;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  afterSubmit?: (certificateId: string) => void;
}

type FormValues = {
  certificate: string;
  privateKey: string;
};

const CertificateUploadModal = ({ certificate, open, onOpenChange, afterSubmit }: CertificateUploadModalProps) => {
  const { t } = useTranslation();
  const { message, notification } = App.useApp();
  const [form] = Form.useForm<FormValues>();

  const fieldCertificate = Form.useWatch("certificate", { form, preserve: true });
  const subjectAltNames = useMemo(() => getCertificateSubjectAltNames(fieldCertificate || "").join(";"), [fieldCertificate]);

  const schema = z.object({
    certificate: z
      .string()
      .nonempty()
      .refine((value) => validatePEMCertificate(value), t("certificate.action.upload.modal.form.certificate.errmsg.invalid")),
    privateKey: z
      .string()
      .nonempty()
      .refine((value) => validatePEMPrivateKey(value), t("certificate.action.upload.modal.form.private_key.errmsg.invalid")),
  });
  const formRule = createSchemaFieldRule(schema);

  useEffect(() => {
    if (!open) return;
    form.setFieldsValue({
      certificate: certificate?.certificate ?? "",
      privateKey: certificate?.privateKey ?? "",
    });
  }, [certificate, form, open]);

  const handleFinish = async (values: FormValues) => {
    try {
      const resp = await uploadCertificate({ certificateId: certificate?.id, ...values });
      message.success(t("common.text.operation_succeeded"));
      afterSubmit?.(resp.data.certificateId);
    } catch (err) {
      console.error(err);
      notification.error({ title: t("common.text.request_error"), description: unwrapErrMsg(err) });
      throw err;
    }
  };

  return (
    <ModalForm<FormValues>
      form={form}
      layout="vertical"
      modalProps={{ destroyOnHidden: true }}
      open={open}
      preserve={false}
      title={t(certificate ? "certificate.action.update.modal.title" : "certificate.action.upload.modal.title")}
      width={640}
      onFinish={handleFinish}
      onOpenChange={onOpenChange}
    >
      <Form.Item label={t("certificate.action.upload.modal.form.name.label")}>
        <Input readOnly value={subjectAltNames} variant="filled" />
      </Form.Item>
      <Form.Item name="certificate" label={t("certificate.action.upload.modal.form.certificate.label")} rules={[formRule]}>
        <FileTextInput autoSize={{ minRows: 4, maxRows: 10 }} accept=".pem,.crt,.cer" />
      </Form.Item>
      <Form.Item name="privateKey" label={t("certificate.action.upload.modal.form.private_key.label")} rules={[formRule]}>
        <FileTextInput autoSize={{ minRows: 4, maxRows: 10 }} accept=".pem,.key" />
      </Form.Item>
    </ModalForm>
  );
};

export default CertificateUploadModal;
