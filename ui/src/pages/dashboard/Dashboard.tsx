import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import {
  IconAlertHexagon,
  IconArrowUpRight,
  IconBox,
  IconCertificate,
  IconCirclePlus,
  IconExternalLink,
  IconHexagonLetterX,
  IconHistory,
  IconLock,
  IconPlugConnected,
  IconReload,
  IconRoute,
  IconShieldCheckered,
  IconUpload,
} from "@tabler/icons-react";
import { useRequest } from "ahooks";
import { App, Button, Card, Col, Row, Skeleton, Table, type TableProps, Typography, theme } from "antd";
import dayjs from "dayjs";
import { ClientResponseError } from "pocketbase";

import { get as getStatistics } from "@/api/statistics";
import Empty from "@/components/Empty";
import CertificateUploadModal from "@/components/certificate/CertificateUploadModal";
import WorkflowRunDetailDrawer from "@/components/workflow/WorkflowRunDetailDrawer";
import WorkflowStatus from "@/components/workflow/WorkflowStatus";
import { type Statistics } from "@/domain/statistics";
import { type WorkflowRunModel } from "@/domain/workflowRun";
import { get as getWorkflowRun, list as listWorkflowRuns } from "@/repository/workflowRun";
import { mergeCls } from "@/utils/css";
import { unwrapErrMsg } from "@/utils/error";

const Dashboard = () => {
  const { t } = useTranslation();

  return (
    <div className="px-4 py-5 md:px-6 lg:px-8">
      <div className="container max-w-screen-2xl">
        <h1 className="mb-6">{t("dashboard.page.heading")}</h1>

        <section>
          <StatisticCards />
        </section>

        <section className="mt-7">
          <SectionHeading title={t("dashboard.shortcut")} />
          <Shortcuts />
        </section>

        <section className="mt-7">
          <Card
            className="overflow-hidden shadow-sm"
            title={<SectionHeading title={t("dashboard.recent_workflow_runs")} />}
            styles={{ header: { minHeight: 56 }, body: { padding: 0 } }}
          >
            <WorkflowRunHistoryTable />
          </Card>
        </section>
      </div>
    </div>
  );
};

const SectionHeading = ({ title }: { title: React.ReactNode }) => (
  <div className="flex items-center gap-3">
    <span className="h-4 w-1 rounded-full bg-primary" />
    <h3 className="m-0 text-base font-semibold">{title}</h3>
  </div>
);

const StatisticCard = ({
  className,
  style,
  label,
  loading,
  icon,
  accent,
  value,
  onClick,
}: {
  className?: string;
  style?: React.CSSProperties;
  label: React.ReactNode;
  loading?: boolean;
  icon: React.ReactNode;
  accent: string;
  value?: string | number | React.ReactNode;
  onClick?: () => void;
}) => {
  return (
    <Card
      className={mergeCls("size-full overflow-hidden border shadow-sm transition-all hover:-translate-y-0.5 hover:shadow-md", className)}
      style={{ ...style, borderTop: `2px solid ${accent}` }}
      styles={{ body: { padding: 0 } }}
      hoverable
      loading={loading}
      onClick={onClick}
    >
      <div className="relative overflow-hidden p-5">
        <div className="pointer-events-none absolute inset-0 z-0 opacity-[0.035]">
          <div
            className="size-full"
            style={{
              backgroundImage: "linear-gradient(currentColor 1px, transparent 1px), linear-gradient(90deg, currentColor 1px, transparent 1px)",
              backgroundSize: "16px 16px",
            }}
          />
        </div>
        <div className="relative flex items-start justify-between gap-4">
          <div>
            <div className="mb-3 truncate text-xs font-medium tracking-wide text-secondary">{label}</div>
            <div className="truncate font-mono text-4xl font-semibold leading-none">{value}</div>
          </div>
          <div
            className="flex size-11 shrink-0 items-center justify-center rounded-lg"
            style={{ background: `color-mix(in srgb, ${accent} 12%, transparent)`, color: accent }}
          >
            {icon}
          </div>
        </div>
      </div>
    </Card>
  );
};

const StatisticCards = ({ className, style }: { className?: string; style?: React.CSSProperties }) => {
  const navigate = useNavigate();

  const { t } = useTranslation();

  const { notification } = App.useApp();

  const cardGridSpans = {
    xs: { flex: "100%" },
    sm: { flex: "50%" },
    md: { flex: "50%" },
    lg: { flex: "25%" },
  };

  const [statistics, setStatistics] = useState<Statistics>();

  const { loading } = useRequest(
    () => {
      return getStatistics();
    },
    {
      onSuccess: (res) => {
        setStatistics(res.data);
      },
      onError: (err) => {
        if (err instanceof ClientResponseError && err.isAbort) {
          return;
        }

        console.error(err);
        notification.error({ title: t("common.text.request_error"), description: unwrapErrMsg(err) });

        throw err;
      },
    }
  );

  return (
    <div className={className} style={style}>
      <Row className="justify-stretch" gutter={[16, 16]}>
        <Col className="overflow-hidden" {...cardGridSpans}>
          <StatisticCard
            accent="var(--color-info)"
            icon={<IconShieldCheckered size={24} />}
            label={t("dashboard.statistics.all_certificates")}
            loading={loading}
            value={statistics?.certificateTotal ?? "-"}
            onClick={() => navigate("/certificates")}
          />
        </Col>
        <Col className="overflow-hidden" {...cardGridSpans}>
          <StatisticCard
            accent="var(--color-primary)"
            icon={<IconRoute size={24} />}
            label={t("dashboard.statistics.all_workflows")}
            loading={loading}
            value={statistics?.workflowTotal ?? "-"}
            onClick={() => navigate("/workflows")}
          />
        </Col>
        <Col className="overflow-hidden" {...cardGridSpans}>
          <StatisticCard
            accent="var(--color-warning)"
            icon={<IconAlertHexagon size={24} />}
            label={t("dashboard.statistics.expiring_soon_certificates")}
            loading={loading}
            value={statistics?.certificateExpiringSoon ?? "-"}
            onClick={() => navigate("/certificates?state=expiringSoon")}
          />
        </Col>
        <Col className="overflow-hidden" {...cardGridSpans}>
          <StatisticCard
            accent="var(--color-error)"
            icon={<IconHexagonLetterX size={24} />}
            label={t("dashboard.statistics.expired_certificates")}
            loading={loading}
            value={statistics?.certificateExpired ?? "-"}
            onClick={() => navigate("/certificates?state=expired")}
          />
        </Col>
      </Row>
    </div>
  );
};

const Shortcuts = ({ className, style }: { className?: string; style?: React.CSSProperties }) => {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { token: themeToken } = theme.useToken();
  const [uploadModalOpen, setUploadModalOpen] = useState(false);

  const actions = [
    {
      key: "upload-certificate",
      label: t("dashboard.shortcut.upload_certificate"),
      icon: <IconUpload size={22} />,
      color: "var(--color-success)",
      onClick: () => setUploadModalOpen(true),
    },
    {
      key: "create-workflow",
      label: t("dashboard.shortcut.create_workflow"),
      icon: <IconCirclePlus size={22} />,
      color: "var(--color-primary)",
      onClick: () => navigate("/workflows/new"),
    },
    {
      key: "change-account",
      label: t("dashboard.shortcut.change_account"),
      icon: <IconLock size={22} />,
      color: "var(--color-warning)",
      onClick: () => navigate("/settings/account"),
    },
    {
      key: "configure-ca",
      label: t("dashboard.shortcut.configure_ca"),
      icon: <IconPlugConnected size={22} />,
      color: "var(--color-info)",
      onClick: () => navigate("/settings/ssl-provider"),
    },
  ];

  return (
    <div className={className} style={style}>
      <Row gutter={[12, 12]}>
        {actions.map((action) => (
          <Col key={action.key} xs={24} sm={12} lg={6}>
            <Card className="group cursor-pointer shadow-sm transition-all hover:-translate-y-0.5 hover:shadow-md" hoverable onClick={action.onClick}>
              <div className="flex items-center gap-3">
                <div
                  className="flex size-10 shrink-0 items-center justify-center rounded-lg"
                  style={{ background: `color-mix(in srgb, ${action.color} 12%, transparent)`, color: action.color }}
                >
                  {action.icon}
                </div>
                <div className="min-w-0 flex-1 truncate text-sm font-medium">{action.label}</div>
                <IconArrowUpRight className="opacity-40 transition-opacity group-hover:opacity-100" color={themeToken.colorTextSecondary} size={18} />
              </div>
            </Card>
          </Col>
        ))}
      </Row>

      <CertificateUploadModal open={uploadModalOpen} onOpenChange={setUploadModalOpen} />
    </div>
  );
};

const WorkflowRunHistoryTable = ({ className, style }: { className?: string; style?: React.CSSProperties }) => {
  const navigate = useNavigate();

  const { t } = useTranslation();

  const { notification } = App.useApp();

  const [tableData, setTableData] = useState<WorkflowRunModel[]>([]);
  const tableColumns: TableProps<WorkflowRunModel>["columns"] = [
    {
      key: "$index",
      align: "center",
      fixed: "left",
      width: 48,
      render: (_, __, index) => index + 1,
    },
    {
      key: "id",
      title: "ID",
      width: 160,
      render: (_, record) => <span className="font-mono">{record.id}</span>,
    },
    {
      key: "workflow",
      title: t("workflow_run.props.workflow"),
      render: (_, record) => {
        const workflow = record.expand?.workflowRef;
        return (
          <div className="max-w-full truncate">
            <Typography.Link
              ellipsis
              onClick={() => {
                if (workflow) {
                  navigate(`/workflows/${workflow.id}`);
                }
              }}
            >
              {workflow?.name ?? <span className="font-mono">{`#${record.workflowRef}`}</span>}
            </Typography.Link>
          </div>
        );
      },
    },
    {
      key: "status",
      title: t("workflow_run.props.status"),
      render: (_, record) => {
        return <WorkflowStatus type="filled" value={record.status} />;
      },
    },
    {
      key: "startedAt",
      title: t("workflow_run.props.started_at"),
      ellipsis: true,
      render: (_, record) => {
        if (record.startedAt) {
          return dayjs(record.startedAt).format("YYYY-MM-DD HH:mm:ss");
        }

        return <></>;
      },
    },
    {
      key: "endedAt",
      title: t("workflow_run.props.ended_at"),
      ellipsis: true,
      render: (_, record) => {
        if (record.endedAt) {
          return dayjs(record.endedAt).format("YYYY-MM-DD HH:mm:ss");
        }

        return <></>;
      },
    },
    {
      key: "artifacts",
      title: t("workflow_run.props.artifacts"),
      width: 160,
      render: (_, record) => {
        if (record.outputs && record.outputs.length > 0) {
          const keys = new Set<string>();
          const icons: React.ReactNode[] = [];

          for (const output of record.outputs) {
            if (output.type === "ref" && output.value?.split("#")?.at(0) === "certificate") {
              const KEY = "certificate";
              if (keys.has(KEY)) continue;

              keys.add(KEY);
              icons.push(<IconCertificate key={KEY} size="1.25em" />);
            } else {
              const KEY = "other";
              if (keys.has(KEY)) continue;

              keys.add(KEY);
              icons.push(<IconBox key={KEY} size="1.25em" />);
            }
          }

          return <div className="flex items-center gap-2">{icons}</div>;
        }

        return <></>;
      },
    },
  ];

  const {
    loading,
    error: loadError,
    run: refreshData,
  } = useRequest(
    () => {
      return listWorkflowRuns({
        page: 1,
        perPage: 15,
        expand: true,
      });
    },
    {
      onSuccess: (res) => {
        setTableData(res.items);
      },
      onError: (err) => {
        if (err instanceof ClientResponseError && err.isAbort) {
          return;
        }

        console.error(err);
        notification.error({ title: t("common.text.request_error"), description: unwrapErrMsg(err) });

        throw err;
      },
    }
  );

  const handleReloadClick = () => {
    if (loading) return;

    refreshData();
  };

  const { drawerProps: detailDrawerProps, ...detailDrawer } = WorkflowRunDetailDrawer.useDrawer();

  const handleRecordDetailClick = (workflowRun: WorkflowRunModel) => {
    const drawer = detailDrawer.open({ data: workflowRun, loading: true });
    getWorkflowRun(workflowRun.id).then((data) => {
      drawer.safeUpdate({ data, loading: false });
    });
  };

  return (
    <div className={className} style={style}>
      <Table<WorkflowRunModel>
        columns={tableColumns}
        dataSource={tableData}
        loading={loading}
        locale={{
          emptyText: loading ? (
            <Skeleton />
          ) : (
            <Empty
              className="py-24"
              title={loadError ? t("common.text.nodata_failed") : t("common.text.nodata")}
              description={loadError ? unwrapErrMsg(loadError) : t("dashboard.recent_workflow_runs.nodata.description")}
              icon={<IconHistory size={24} />}
              extra={
                loadError ? (
                  <Button ghost icon={<IconReload size="1.25em" />} type="primary" onClick={handleReloadClick}>
                    {t("common.button.reload")}
                  </Button>
                ) : (
                  <Button icon={<IconExternalLink size="1.25em" />} type="primary" onClick={() => navigate("/workflows")}>
                    {t("dashboard.recent_workflow_runs.nodata.button")}
                  </Button>
                )
              }
            />
          ),
        }}
        pagination={false}
        rowClassName="cursor-pointer"
        rowKey={(record) => record.id}
        scroll={{ x: "max(100%, 720px)" }}
        onRow={(record) => ({
          onClick: () => {
            handleRecordDetailClick(record);
          },
        })}
      />

      <WorkflowRunDetailDrawer {...detailDrawerProps} />
    </div>
  );
};

export default Dashboard;
