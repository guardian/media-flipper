import React from 'react';
import PropTypes from 'prop-types';
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";

class BatchStatusSummary extends React.Component {
    static propTypes = {
        batchStatus: PropTypes.object.isRequired,
        className: PropTypes.string
    };

    render() {
        const finalClassName = this.props.className ? "status-summary-container " + this.props.className : "status-summary-container";
        return <ul className={finalClassName}>
            <li className="status-summary-entry"><FontAwesomeIcon icon="pause-circle" className="inline-icon" style={{color: "darkblue"}}/>{this.props.batchStatus.pendingCount} items pending</li>
            <li className="status-summary-entry"><FontAwesomeIcon icon="play-circle" className="inline-icon" style={{color: "darkgreen"}}/>{this.props.batchStatus.activeCount} items active</li>
            <li className="status-summary-entry"><FontAwesomeIcon icon="check-circle" className="inline-icon" style={{color: "darkgreen"}}/>{this.props.batchStatus.completedCount} items completed</li>
            <li className="status-summary-entry"><FontAwesomeIcon icon="times-circle" className="inline-icon" style={{color: "darkred"}}/>{this.props.batchStatus.errorCount} items failed</li>
        </ul>
    }
}

export default BatchStatusSummary;
