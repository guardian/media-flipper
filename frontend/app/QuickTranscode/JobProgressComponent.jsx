import React from 'react';
import PropTypes from 'prop-types';

class JobProgressComponent extends React.Component {
    static propTypes = {
        jobStepList: PropTypes.array.isRequired,    //array of objects corresponding to the job steps
        currentJobStep: PropTypes.number.isRequired,
        className: PropTypes.string.isRequired,
        hidden: PropTypes.bool.isRequired
    };

    // constructor(props) {
    //     super(props);
    // }

    render() {
        return <div className={this.props.className} style={{display: this.props.hidden ? "none" : "inherit"}}>
            {
                this.props.jobStepList.map((step,idx)=>{
                    const classList = [
                        "transcode-info-block",
                        idx>=this.props.currentJobStep ? "job-step-unreached" : "job-step-reached"
                    ];

                    return <span className={classList.join(" ")}>{step.InProgressLabel} {idx<this.props.currentJobStep ? "Done!" : ""} {idx>this.props.currentJobStep ? "not started" : ""}</span>
                })
            }
        </div>
    }
}

export default JobProgressComponent;