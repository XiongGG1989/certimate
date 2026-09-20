import { type CertificateFormatType } from "@/domain/certificate";

import { post as httpPost } from "./_api";

export const download = (
  certificateId: string,
  format?: CertificateFormatType,
  params?: { pfxPassword?: string; pfxEncoder?: string; jksAlias?: string; jksKeypass?: string; jksStorepass?: string }
) => {
  type RespData = {
    zipBytes: string;
  };

  return httpPost<RespData>({
    url: `/api/certificates/${encodeURIComponent(certificateId)}/download`,
    body: {
      fileFormat: format,
      ...params,
    },
  });
};

export const revoke = (certificateId: string) => {
  return httpPost({
    url: `/api/certificates/${encodeURIComponent(certificateId)}/revoke`,
  });
};

export const upload = (params: { certificateId?: string; certificate: string; privateKey: string }) => {
  type RespData = {
    certificateId: string;
  };

  const path = params.certificateId ? `/${encodeURIComponent(params.certificateId)}` : "";
  return httpPost<RespData>({
    url: `/api/certificates${path}/upload`,
    body: {
      certificate: params.certificate,
      privateKey: params.privateKey,
    },
  });
};
