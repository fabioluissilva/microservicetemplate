package commonscheduler

import (
	"fmt"

	"github.com/fabioluissilva/microservicetemplate/commonconfig"
	"github.com/fabioluissilva/microservicetemplate/commonlogger"
	"github.com/fabioluissilva/microservicetemplate/commonmetrics"
	"github.com/go-co-op/gocron/v2"
)

var scheduler gocron.Scheduler

type CronJob struct {
	Name     string   `json:"name"`
	CronExpr string   `json:"cron_expr"`
	Job      func()   `json:"-"`
	Tags     []string `json:"tags"`
}

var jobs []CronJob

type JobInfo struct {
	Name    string   `json:"name"`
	Tags    []string `json:"tags"`
	NextRun string   `json:"next_run"`
}

func GetJobsInfo() []JobInfo {
	var infos []JobInfo
	for _, job := range scheduler.Jobs() {
		nextRun, _ := job.NextRun()
		info := JobInfo{
			Name:    job.Tags()[0], // or use a custom tag for name
			Tags:    job.Tags(),
			NextRun: nextRun.Format("2006-01-02 15:04:05"),
		}
		infos = append(infos, info)
	}
	return infos
}

func Heartbeat() {
	if commonconfig.GetConfig().GetHeartBeatDebug() {
		commonlogger.GetLogger().Debug("Sending Heartbeat...")
	}
	commonmetrics.HeartbeatCount.Inc()
	commonmetrics.HeartbeatMessage.SetToCurrentTime()
}

// RegisterJobs receives a slice of CronJob and appends them to the heartbeat job
func RegisterJobs(extraJobs []CronJob) {
	// Always start with the heartbeat job
	jobs = []CronJob{
		{
			Name:     "heartbeatjob",
			CronExpr: commonconfig.GetConfig().GetHeartBeatCron(),
			Job:      Heartbeat,
			Tags:     []string{"heartbeatjob"},
		},
	}
	// Append any additional jobs
	jobs = append(jobs, extraJobs...)
}

func InitScheduler(extraJobs []CronJob) {
	var err error
	scheduler, err = gocron.NewScheduler()
	if err != nil {
		commonlogger.GetLogger().Error(fmt.Sprintf("InitScheduler: Error creating scheduler: %s", err.Error()))
		return
	}
	commonlogger.GetLogger().Debug("InitScheduler: Registering jobs...")
	RegisterJobs(extraJobs)
	for _, job := range jobs {
		commonlogger.GetLogger().Debug("InitScheduler: Setting Cron for " + job.Name + ": " + job.CronExpr)
		cronJob, err := scheduler.NewJob(
			gocron.CronJob(job.CronExpr, false),
			gocron.NewTask(job.Job),
			gocron.WithTags(job.Tags...),
		)
		if err != nil {
			commonlogger.GetLogger().Error("InitScheduler: Error starting " + job.Name + ": " + err.Error())
			continue
		}
		commonlogger.GetLogger().Debug("InitScheduler: Started " + job.Name + " with ID: " + cronJob.ID().String())
	}
	commonlogger.GetLogger().Debug("InitScheduler: Starting Scheduler...")
	scheduler.Start()
}

func ListGocronJobs() []gocron.Job {
	return scheduler.Jobs()
}

func GetScheduledJobs() []CronJob {
	return jobs
}
