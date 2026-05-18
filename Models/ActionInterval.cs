using System;

namespace AlbionInfoTracker.Models;

public class ActionInterval
{
    public ActionInterval(DateTime startTime)
    {
        StartTime = startTime;
    }

    public DateTime StartTime { get; }
    public DateTime? EndTime { get; set; }
    public TimeSpan TimeSpan => EndTime != null ? (DateTime)EndTime - StartTime : new TimeSpan(0);
}
