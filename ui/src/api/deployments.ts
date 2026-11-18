import { post as httpPost } from "./_api";

export const listUCloudUCDNDomains = async ({
  accessId,
  endpoint,
  filterByCertificate,
  certificateId,
  certificateWorkflowId,
  certificateOutputNodeId,
  subjectAltNames,
}: {
  accessId: string;
  endpoint?: string;
  filterByCertificate?: boolean;
  certificateId?: string;
  certificateWorkflowId?: string;
  certificateOutputNodeId?: string;
  subjectAltNames?: string[];
}) => {
  const resp = await httpPost<{ domains: string[] }>({
    url: "/api/deployment/providers/ucloud-ucdn/domains",
    body: {
      accessId,
      endpoint,
      filterByCertificate,
      certificateId,
      certificateWorkflowId,
      certificateOutputNodeId,
      subjectAltNames,
    },
  });
  return resp.data.domains;
};
